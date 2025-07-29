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
		Gender    string `xml:"gender"`
	} `xml:"row"`
}

func main() {
	http.HandleFunc("/", SearchServer)
	http.ListenAndServe(":8080", nil)
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token != validToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]User{})
		return
	}

	var user UserInfo
	err = xml.Unmarshal(file, &user)
	if err != nil {
		panic(err)
	}

	var users []User
	for _, u := range user.User {
		users = append(users, User{
			Id:     u.Id,
			Name:   u.FirstName + " " + u.LastName,
			Age:    u.Age,
			About:  u.About,
			Gender: u.Gender,
		})
	}

	params := r.URL.Query()
	query := params.Get("query")
	orderField := params.Get("order_field")
	orderBy := params.Get("order_by")
	limit := params.Get("limit")
	offset := params.Get("offset")

	var queryFilter []User
	for _, u := range users {
		if query == "" || strings.Contains(u.Name, query) || strings.Contains(u.About, query) {
			queryFilter = append(queryFilter, u)
		}
	}
	if query != "" && len(queryFilter) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchErrorResponse{
			Error: "unsupported query",
		})
	}

	var order int
	switch orderBy {
	case "-1":
		order = OrderByAsc
	case "0", "":
		order = OrderByAsIs
	case "1":
		order = OrderByDesc
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchErrorResponse{
			Error: ErrorBadOrderField,
		})
		return
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
		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(SearchErrorResponse{
				Error: "Invalid OrderField"})
		}
	}

	limitCount := len(queryFilter)
	if limit != "" {
		l, err := strconv.Atoi(limit)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(SearchErrorResponse{
				Error: "unsupported limit",
			})
			return
		}
		if l == 0 || l < 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(SearchErrorResponse{
				Error: "limit not must be null",
			})
			return
		}
		if l > 0 {
			limitCount = l
		}
	}

	offsetCount := 0
	if offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 && o < len(queryFilter) {
			offsetCount = o
		}
	}

	start := offsetCount
	if start > len(queryFilter) {
		start = len(queryFilter)
	}

	end := start + limitCount
	if end > len(queryFilter) {
		end = len(queryFilter)
	}

	result := queryFilter[start:end]
	if offsetCount < len(result) {
		result = result[offsetCount:]
	} else {
		result = []User{}
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
var searchName = "Boyd Wolf"
var invalidSearchName = "fsfjklsjfsklfjslfjsl"
var searchAbout = "Nulla cillum"
var invalidSearchAbout = "jfksljfklsjfslkfsj"
var userMock = User{
	Id:     0,
	Name:   "Boyd Wolf",
	Age:    22,
	About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
	Gender: "male",
}

func TestSearchUser(t *testing.T) {
	cases := []TestCase{
		{
			ID: "0. success result",
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			Token:   validToken,
			isError: false,
		},
		{
			ID:      "1. without token",
			Token:   " ",
			Request: SearchRequest{},
			isError: true,
		},
		{
			ID:      "2. invalid token",
			Token:   "invalid_token",
			Request: SearchRequest{},
			isError: true,
		},
		{
			ID: "3. limit = -1",
			Request: SearchRequest{
				Limit: -1,
			},
			isError: true,
		},
		{
			ID: "4. offset = 0",
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
			ID: "5. offset = -1",
			Request: SearchRequest{
				Offset: -1,
			},
			isError: true,
		},
		{
			ID: "6. Order by Name",
			Request: SearchRequest{
				OrderBy:    1,
				OrderField: "Name",
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			isError: false,
		},
		{
			ID: "7. Order by Age",
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
			ID: "8. Order by Id",
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
			ID: "9. Check order as is",
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
			ID: "10. Check order by asc",
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
			ID: "11. Check order by desc",
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
			ID: "12. Search by query Name",
			Request: SearchRequest{
				Query: searchName,
				Limit: 1,
			},
			Response: &SearchResponse{
				Users: []User{
					userMock,
				},
				NextPage: false,
			},
			isError: false,
		},
		{
			ID: "13. Search by query About",
			Request: SearchRequest{
				Limit: 1,
				Query: searchAbout,
			},
			Response: &SearchResponse{
				Users: []User{
					userMock,
				},
				NextPage: false,
			},
			isError: false,
		},
		{
			ID: "14. InvalidSearch by query About",
			Request: SearchRequest{
				Query: invalidSearchAbout,
			},
			isError: true,
		},
		{
			ID: "15. InvalidSearch by query Name",
			Request: SearchRequest{
				Query: invalidSearchName,
			},
			isError: true,
		},
		{
			ID:      "16. Empty token",
			Token:   "\x00",
			Request: SearchRequest{},
			isError: true,
		},
		{
			ID: "17. Invalid Order by",
			Request: SearchRequest{
				OrderBy: 10,
			},
			isError: true,
		},
		{
			ID: "18. Invalid OrderField",
			Request: SearchRequest{
				OrderBy:    1,
				OrderField: "Invalid",
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
			AccessToken: token,
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
