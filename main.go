package main

import (
	"github.com/gin-gonic/gin"
	"github.com/garyburd/redigo/redis"
	"github.com/DuoSRX/gokiq"
	"net/http"
	"time"
	"strings"
)

func main() {

	router := gin.Default()
	pool   := &redis.Pool{ Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", ":6379")
		if err != nil {
			return nil, err
		}
		return c, err
	} }

	// get orders for a given shop
	router.POST("v2/shops/:shop_uuid/orders", func(c *gin.Context) {

		// grab the request data to be sent to Redis
		shopUUID      := c.Param("shop_uuid")
		authString    := c.Request.Header.Get( "authorization" )
		auth          := strings.SplitN(strings.Replace(authString, "Token ", "", 1), ",", 2)
		params        := []interface{}{auth, "params_hash_for_worker"}

		// prepare and enqueue job to Redis
		job := gokiq.NewJob("ApiOrderEventWorker", "default", params, 1)

		job.Enqueue(pool)
		job.EnqueueAt(time.Now(), pool)

		// tell the requester that everything's gonna be ok
		c.JSON(http.StatusOK, gin.H{
			"status"         : 200,
			"authentication" : auth,
			"shop_uuid"      : shopUUID,
		})
	})

	router.Run(":8080")
}
