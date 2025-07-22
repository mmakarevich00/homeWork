package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const filePath string = "./dataset.xml"

type UserInfo struct {
	XMLName xml.Name `xml:"root"`
	User    []struct {
		Id        int    `xml:"id"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
		Age       int    `xml:"age"`
		About     string `xml:"about"`
	} `xml:"row"`
}

type UserData struct {
	Id    int
	Name  string
	Age   int
	About string
}

func main() {
	http.HandleFunc("/", SearchServer)
	http.ListenAndServe(":8080", nil)
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	var user UserInfo
	err = xml.Unmarshal(file, &user)
	if err != nil {
		panic(err)
	}

	var users []UserData
	for _, u := range user.User {
		users = append(users, UserData{
			Id:    u.Id,
			Name:  u.FirstName + " " + u.LastName,
			Age:   u.Age,
			About: u.About,
		})
	}

	params := r.URL.Query()
	query := params.Get("query")
	orderField := params.Get("order_field")
	orderBy := params.Get("order_by")
	limit := params.Get("limit")
	offset := params.Get("offset")

	var queryFilter []UserData
	for _, u := range users {
		if query == "" || strings.Contains(u.Name, query) || strings.Contains(u.About, query) {
			queryFilter = append(queryFilter, u)
		}
	}

	var order int
	switch orderBy {
	case "-1":
		order = OrderByAsc
	case "0":
		order = OrderByAsIs
	case "1":
		order = OrderByDesc
	default:
		orderBy = ErrorBadOrderField
	}

	if order != OrderByAsIs {
		switch orderField {
		case "Id":
			sort.Slice(queryFilter, func(i, j int) bool {
				if order == OrderByAsc {
					return queryFilter[i].Id < queryFilter[j].Id
				}
				return queryFilter[i].Id > queryFilter[j].Id
			})
		case "Age":
			sort.Slice(queryFilter, func(i, j int) bool {
				if order == OrderByAsc {
					return queryFilter[i].Age < queryFilter[j].Age
				}
				return queryFilter[i].Age > queryFilter[j].Age
			})
		case "", "Name":
			sort.Slice(queryFilter, func(i, j int) bool {
				if order == OrderByAsc {
					return queryFilter[i].Name < queryFilter[j].Name
				}
				return queryFilter[i].Name > queryFilter[j].Name
			})
		}
	}

	limitCount := len(queryFilter)
	if limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l >= 0 {
			if l < limitCount {
				limitCount = l
			}
		}
	}

	offsetCount := 0
	if offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 && o < len(queryFilter) {
			offsetCount = o
		}

	}

	result := queryFilter
	if offsetCount < len(result) {
		result = result[offsetCount:]
	} else {
		result = []UserData{}
	}
	if limitCount < len(result) {
		result = result[:limitCount]
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		panic(err)
	}
}

type TestCase struct {
	ID       string
	Request  SearchRequest
	Response *SearchResponse
	Token    string
	isError  bool
}

var validToken = "valid_token"
var searchName = "Leanna Travis"
var invalidSearchName = "fsfjklsjfsklfjslfjsl"
var searchAbout = "Lorem"
var invalidSearchAbout = "jfksljfklsjfslkfsj"

func TestSearchUser(t *testing.T) {
	cases := []TestCase{
		{
			ID:      "success result",
			Request: SearchRequest{},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			Token:   validToken,
			isError: false,
		},
		{
			ID:      "without token",
			Token:   "",
			Request: SearchRequest{},
			isError: true,
		},
		{
			ID:      "invalid token",
			Token:   "invalid_token",
			Request: SearchRequest{},
			isError: true,
		},
		{
			ID: "limit = 0",
			Request: SearchRequest{
				Limit: 0,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: false,
			},
			isError: false,
		},
		{
			ID: "limit = 25",
			Request: SearchRequest{
				Limit: 25,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "limit = -1",
			Request: SearchRequest{
				Limit: -1,
			},
			isError: true,
		},
		{
			ID: "offset = 0",
			Request: SearchRequest{
				Offset: 0,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "offset = 5",
			Request: SearchRequest{
				Offset: 5,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "offset = -1",
			Request: SearchRequest{
				Offset: -1,
			},
			isError: true,
		},
		{
			ID: "Order by Name",
			Request: SearchRequest{
				Query:      searchName,
				OrderBy:    0,
				OrderField: "Name",
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "Order by Age",
			Request: SearchRequest{
				OrderBy:    0,
				OrderField: "Age",
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "Order by Id",
			Request: SearchRequest{
				OrderBy:    0,
				OrderField: "Id",
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "Check order as is",
			Request: SearchRequest{
				OrderBy:    0,
				OrderField: "Id",
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "Check order by asc",
			Request: SearchRequest{
				OrderField: "Id",
				OrderBy:    -1,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "Check order by desc",
			Request: SearchRequest{
				OrderField: "Id",
				OrderBy:    1,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "Search by query Name",
			Request: SearchRequest{
				Query: searchName,
			},
			Response: &SearchResponse{
				Users: []User{
					{
						Name: searchName,
					},
				},
				NextPage: false,
			},
			isError: false,
		},
		{
			ID: "Search by query About",
			Request: SearchRequest{
				Query: searchAbout,
			},
			Response: &SearchResponse{
				Users: []User{
					{
						About: searchAbout,
					},
				},
				NextPage: false,
			},
			isError: false,
		},
		{
			ID: "InvalidSearch by query About",
			Request: SearchRequest{
				Query: invalidSearchAbout,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: false,
			},
			isError: true,
		},
		{
			ID: "InvalidSearch by query Name",
			Request: SearchRequest{
				Query: invalidSearchName,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: false,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		token := item.Token
		if token == "" {
			token = validToken
		}
		c := SearchClient{
			AccessToken: validToken,
			URL:         ts.URL,
		}
		result, err := c.FindUsers(item.Request)

		if err != nil && !item.isError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.isError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Response, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Response, result)
		}
	}
	ts.Close()
}
