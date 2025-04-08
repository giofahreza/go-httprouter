package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type User struct {
	Name     string `json:"name" required:"true" min:"3" max:"10"`
	Age      int    `json:"age" required:"true" min:"25" max:"50"`
	Email    string `json:"email" required:"true" min:"10" max:"100"`
	Password string `json:"password" required:"true" min:"3" max:"10"`
}

type Product struct {
	Name  string `json:"name" required:"true" min:"3" max:"10"`
	Price int    `json:"price" required:"true" min:"1000000" max:"100000000"`
	Stock int    `json:"stock" required:"true" min:"1" max:"100"`
}

type NewProductResponse struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type EndpointResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func EndpointResponses(w http.ResponseWriter, response EndpointResponse) {
	w.Header().Set("Content-Type", "application/json")
	if response.Code != 200 {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(response)
}

func ValidateStruct(s interface{}) error {
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", s)
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		value := val.Field(i)

		if field.Tag.Get("required") == "true" && (value.String() == "" || (value.Kind() == reflect.Int && value.Int() <= 0)) {
			return fmt.Errorf("%s is required", field.Name)
		}

		min, err := strconv.Atoi(field.Tag.Get("min"))
		max, err := strconv.Atoi(field.Tag.Get("max"))
		if err != nil {
			return fmt.Errorf("invalid min/max value for %s: %v", field.Name, err)
		}

		if min > 0 || max > 0 {
			// if value string
			if value.Kind() == reflect.String {
				if value.Len() < min {
					return fmt.Errorf("%s must be at least %s characters long", field.Name, min)
				}
				if value.Len() > max {
					return fmt.Errorf("%s must be at most %s characters long", field.Name, max)
				}
			}

			// if value int
			if value.Kind() == reflect.Int {
				if value.Int() < int64(min) {
					return fmt.Errorf("%s must be at least %d", field.Name, min)
				}
				if value.Int() > int64(max) {
					return fmt.Errorf("%s must be at most %d", field.Name, max)
				}
			}
		}
	}

	return nil
}

func main() {
	router := httprouter.New()
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		log.Println("Panic:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		panic("Simulated panic")
		w.Write([]byte("Server up"))
	})

	// User Endpoint
	router.GET("/user/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id := ps.ByName("id")
		w.Write([]byte("User ID: " + id))
	})
	router.POST("/user", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		log.Print(r.FormValue("name"))

		user := User{
			Name: r.FormValue("name"),
			Age: func() int {
				age, _ := strconv.Atoi(r.FormValue("age"))
				return age
			}(),
			Email:    r.FormValue("email"),
			Password: r.FormValue("password"),
		}

		err = ValidateStruct(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write([]byte("User created successfully"))
	})

	// Product Endpoint
	router.POST("/product", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var product Product
		err := json.NewDecoder(r.Body).Decode(&product)
		if err != nil {
			log.Print("JSON Decoding err : ", err)
			EndpointResponses(w, EndpointResponse{
				Code: 400,
				Msg:  "Invalid JSON",
				Data: nil,
			})
			return
		}

		err = ValidateStruct(product)
		if err != nil {
			log.Print("Validation err : ", err)
			EndpointResponses(w, EndpointResponse{
				Code: 400,
				Msg:  err.Error(),
				Data: nil,
			})
			return
		}

		EndpointResponses(w, EndpointResponse{
			Code: 200,
			Msg:  "Product created successfully",
			Data: NewProductResponse{
				Name:  product.Name,
				Price: product.Price,
			},
		})
	})

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
