package main

import (
	"github.com/gin-gonic/gin"
	"github.com/garyburd/redigo/redis"
	"net/http"
	"time"
	"strings"
	"fmt"
	"regexp"
	"github.com/aws/aws-lambda-go/lambda"
)

func ParseTokenAndOptions(auth string) string {

	_, err := regexp.MatchString("^Token ", auth)
	if err != nil {
		return auth
	}

	return strings.SplitN(strings.Replace( strings.Replace(auth, "Token ", "", 1), "token=", "", 1), " ", 2)[0]
}

func handler() {

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

		// grab auth headers
		var authArray [2]string
		authString   := c.Request.Header.Get("authorization")
		authArray[0]  = strings.SplitN(authString," ", 2)[0] // e.g. "Bearer"
		authArray[1]  = ParseTokenAndOptions(authString)           // the auth token

		// build request data arguments
		requestData := make( map[string]string )

		requestData["requested_at"]    = time.Now().String()
		requestData["request_method"]  = c.Request.Method
		requestData["request_path"]    = c.Request.Header.Get("REQUEST_PATH")
		requestData["request_referer"] = c.Request.Referer()
		requestData["user_agent"]      = c.Request.UserAgent()
		requestData["remote_ip"]       = c.Request.RemoteAddr
		requestData["auth_header"]     = c.Request.Header.Get("HTTP_AUTHORIZATION")
		requestData["request_id"]      = shopUUID
		requestData["release_sha"]     = ""
		requestData["shop_domain"]     = c.Request.Header.Get("x-jilt-shop-domain")

		// what the data should look like in there:
		//{"updated_at"=>"2018-04-27T20:28:12.627999Z", "cart_token"=>"f3b8bbbd-beb1-4a1f-ba68-c98159e88f58", "checkout_url"=>"https://jilt-api-docs.example.com", "line_items"=>[{"key"=>"123"}], "shop_id"=>"1", "order"=>{"checkout_url"=>"https://jilt-api-docs.example.com", "cart_token"=>"f3b8bbbd-beb1-4a1f-ba68-c98159e88f58"}, "domain_header"=>nil}, "create", nil
		params := []interface{}{
			authArray,
			requestData,
		}

		fmt.Println("Sending new task to Redis...")

		// prepare and enqueue job to Redis
		job := NewJob("ApiOrderEventWorker", "default", params, 1)

		fmt.Printf("Got Job ID %s\n", job.JID)

		job.Enqueue(pool)
		job.EnqueueAt(time.Now(), pool)

		fmt.Println("Done!")

		// tell the requester that everything's gonna be ok
		c.JSON(http.StatusOK, gin.H{ "status" : 200 })
	})

	router.Run(":8080")
}

func main() {
	lambda.Start(handler)
}