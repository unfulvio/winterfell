package main

import (
	"github.com/gin-gonic/gin"
	"github.com/garyburd/redigo/redis"
	"github.com/DuoSRX/gokiq"
	"net/http"
	"time"
	"strings"
	"fmt"
)

func main() {

	fmt.Println( "-- Starting Wintefell --" )

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

		shopUUID := c.Param("shop_uuid")

		// return server error if pool errored out
		if pool == nil {
			c.JSON(500, gin.H{ "status" : 500 })
		}

		fmt.Printf( "Requesting orders for shop %s...\n", shopUUID )

		// grab the request data to be sent to Redis
		authString := c.Request.Header.Get( "authorization" )
		auth       := strings.SplitN(strings.Replace(authString, "Token ", "", 1), ",", 2)
		params     := []interface{}{auth, "params_hash_for_worker"}

		fmt.Println( "Sending new task to Redis..." )

		// prepare and enqueue job to Redis
		job := gokiq.NewJob("ApiOrderEventWorker", "default", params, 1)

		jobID := job.Enqueue(pool)

		fmt.Printf( "Got Job ID %s\n", jobID )

		job.EnqueueAt(time.Now(), pool)

		fmt.Println( "Done!" )

		// tell the requester that everything's gonna be ok
		c.JSON(http.StatusOK, gin.H{ "status" : 200 })
	})

	router.Run(":8080")
}
