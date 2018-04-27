package main

import (
	"github.com/gin-gonic/gin"
	"strings"
	"github.com/garyburd/redigo/redis"
	"net/http"
	"time"
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
		params        := "params_hash_for_worker"
		someInterface := []{
			Authorization: auth,
			Params:        params
		}

		// prepare and enqueue job to Redis
		job := NewJob(
			"ApiOrderEventWorker",
			"default",
			someInterface,
			1)

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
