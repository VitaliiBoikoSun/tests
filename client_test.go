package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const filePath = "./dataset.xml"

type TestCase struct {
	Request  SearchRequest
	Response *SearchResponse
	IsError  bool
}

type XMLUser struct {
	ID        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type JSONUser struct {
	ID     int    `json:"id"`
	Age    int    `json:"age"`
	Name   string `json:"name"`
	About  string `json:"about"`
	Gender string `json:"gender"`
}

type Users struct {
	List []XMLUser `xml:"row"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token != "token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	query := r.FormValue("query")
	orderField := r.FormValue("order_field")
	if orderField != "" && orderField != "Id" && orderField != "Age" && orderField != "Name" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"ErrorBadOrderField\"}"))
		return
	}
	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"ErrorBadOrderField\"}"))
		return
	}
	/*limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"ErrorBadLimitField\"}"))
		return
	}*/
	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"ErrorBadOffsetField\"}"))
		return
	}

	if query == "request_error" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if query == "internal_err" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if query == "bad_request" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if (orderBy > 1) || (orderBy < -1) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"ErrorBadOrderField\"}"))
		return
	}

	if query == "bad_request_json" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"\"}"))
		return
	}

	if query == "bad_response_json" {
		w.Write([]byte("{\"error\":\""))
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	xmlData, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var users Users
	err = xml.Unmarshal(xmlData, &users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	offseted := 0
	items := make([]*JSONUser, 0)
	for _, item := range users.List {
		if offset > offseted {
			offseted++
			continue
		}

		fullName := item.FirstName + item.LastName
		if query != "" && !(strings.Contains(fullName, query) || strings.Contains(item.About, query)) {
			continue
		}

		items = append(items, &JSONUser{
			ID:     item.ID,
			Name:   item.FirstName + " " + item.LastName,
			Age:    item.Age,
			About:  item.About,
			Gender: item.Gender,
		})
	}

	if orderField != "" && orderBy != OrderByAsIs {
		sort.Slice(items, func(i, j int) bool {
			switch orderField {
			case "Name":
				if orderBy == OrderByDesc {
					return items[i].Name < items[j].Name
				}
				return items[i].Name > items[j].Name

			case "Id":
				if orderBy == OrderByDesc {
					return items[i].ID < items[j].ID
				}
				return items[i].ID > items[j].ID

			case "Age":
				if orderBy == OrderByDesc {
					return items[i].Age < items[j].Age
				}
				return items[i].Age > items[j].Age

			default:
				panic("unknown field")
			}
		})
	}
	responseString, err := json.Marshal(items)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"Can't marshal response\"}"))
		return
	}
	//fmt.Printf("%#v", string(responseString))
	fmt.Fprintln(w, string(responseString))
}

func TestSearchClientFindUserUnauthorized(t *testing.T) {

	cases := []TestCase{
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     2,
				Query:      "",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: "",
	}

	for caseNum, item := range cases {
		result, err := searchClient.FindUsers(item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Response, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Response, result)
		}
	}
}

func TestSearchClientFindUserReqError(t *testing.T) {

	cases := []TestCase{
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     2,
				Query:      "",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         "",
		AccessToken: "",
	}

	for caseNum, item := range cases {
		result, err := searchClient.FindUsers(item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Response, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Response, result)
		}
	}
}

func TestSearchClientFindUserFails(t *testing.T) {

	cases := []TestCase{
		TestCase{
			Request: SearchRequest{
				Limit:      -1,
				Offset:     2,
				Query:      "",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      26,
				Offset:     -1,
				Query:      "",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "internal_err",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "bad_request",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "bad_order_field",
				OrderBy:    10,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "bad_request_json",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "bad_response_json",
				OrderBy:    OrderByAsc,
				OrderField: "",
			},
			Response: nil,
			IsError:  true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: "token",
	}

	for caseNum, item := range cases {
		result, err := searchClient.FindUsers(item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Response, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Response, result)
		}
	}
}

func TestSearchClientFindUser(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Boyd",
				OrderBy:    OrderByAsIs,
				OrderField: "",
			},
			Response: &SearchResponse{
				NextPage: false,
				Users: []User{
					User{
						Id:     0,
						Name:   "Boyd Wolf",
						Age:    22,
						About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
						Gender: "male",
					},
				},
			},
			IsError: false,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Dillard",
				OrderBy:    OrderByAsIs,
				OrderField: "",
			},
			Response: &SearchResponse{
				NextPage: true,
				Users: []User{
					User{
						Id:     3,
						Name:   "Everett Dillard",
						Age:    27,
						About:  "Sint eu id sint irure officia amet cillum. Amet consectetur enim mollit culpa laborum ipsum adipisicing est laboris. Adipisicing fugiat esse dolore aliquip quis laborum aliquip dolore. Pariatur do elit eu nostrud occaecat.\n",
						Gender: "male",
					},
				},
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: "token",
	}
	for caseNum, item := range cases {
		result, err := searchClient.FindUsers(item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Response, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Response, result)
		}
	}

}

func TimeoutServer(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second)
}

func TestSearchClientFindUserReqTimeout(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Request:  SearchRequest{},
			Response: nil,
			IsError:  true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(TimeoutServer))
	ts.Config.WriteTimeout = time.Millisecond
	defer ts.Close()

	searchClient := SearchClient{
		URL: ts.URL,
	}

	for caseNum, item := range cases {
		result, err := searchClient.FindUsers(item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Response, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Response, result)
		}
	}
}
