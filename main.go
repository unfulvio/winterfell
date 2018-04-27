package main

import (
	"github.com/gin-gonic/gin"
	"github.com/garyburd/redigo/redis"
	"net/http"
	"time"
	"strings"
	"fmt"
)

func main() {

	fmt.Println("-- Starting Wintefell --")

	router := gin.Default()
	pool   := &redis.Pool{ Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", "127.0.0.1:6379")
		if err != nil {
			return nil, err
		}
		return c, err
	} }

	// get orders for a given shop
	router.POST("v2/shops/:shop_uuid/orders", func(c *gin.Context) {

		// return server error if pool errored out
		if pool == nil {
			c.JSON(500, gin.H{ "status" : 500 })
			fmt.Println("An error occurred: could not connect to Redis.")
			return
		}

		// grab the shop ID
		shopUUID := c.Param("shop_uuid")

		fmt.Printf("Requesting orders for shop %s...\n", shopUUID)

		// grab other request data to send to Redis
		authString := c.Request.Header.Get( "authorization" )
		auth       := strings.SplitN(strings.Replace(authString, "Token ", "", 1), ",", 2)
		params     := []interface{}{auth, "params_hash_for_worker"}

		fmt.Println("Sending new task to Redis...")

		// prepare and enqueue job to Redis
		job := NewJob("ApiOrderEventWorker", "default", params, 1)

		jobID := job.Enqueue(pool)

		// looks like GoKiq doesn't return a job ID so this might be null after all...
		fmt.Printf("Got Job ID %s\n", jobID)

		job.EnqueueAt(time.Now(), pool)

		fmt.Println("Done!")

		// tell the requester that everything's gonna be ok
		c.JSON(http.StatusOK, gin.H{ "status" : 200 })
	})

	router.Run(":8080")
}
