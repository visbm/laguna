package main

import (
	"fmt"
	"log"

	"github.com/valyala/fasthttp"
)

func main() {
	// Создаём объект запроса и ответа
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	//defer fasthttp.ReleaseResponse(resp)

	// Настраиваем запрос
	req.SetRequestURI("https://jsonplaceholder.typicode.com/comments/1")
	req.Header.SetMethod("GET")

	// Выполняем запрос
	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	respBytes := make([]byte, len(resp.Body()))
	copy(respBytes, resp.Body())
	fasthttp.ReleaseResponse(resp)

	fmt.Println("Body preview:")
	fmt.Println(string(respBytes))

	fmt.Println("----------------")
	// Настраиваем запрос
	req.SetRequestURI("https://jsonplaceholder.typicode.com/posts/1")
	req.Header.SetMethod("GET")

	if err := client.Do(req, resp); err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	fmt.Println("Body preview:")
	fmt.Println(string(respBytes))
	fmt.Println("----------------")
	fmt.Println("Body preview:")
	fmt.Println(string(resp.Body()))

}
