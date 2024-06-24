package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
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

/*
go test -v -cover -coverprofile cover.out
go tool cover -html=cover.out
*/

const AccessToken = "abc123"

type XmlData struct {
	XMLName xml.Name `xml:"root"`
	Rows    []XmlRow `xml:"row"`
}

type XmlRow struct {
	XMLName   xml.Name `xml:"row"`
	Id        int      `xml:"id"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	About     string   `xml:"about"`
	Age       int      `xml:"age"`
	Gender    string   `xml:"gender"`
}

type testCase struct {
	request *SearchRequest
	result  *SearchResponse
	errMsg  string
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") != AccessToken {
		http.Error(w, "Invalid access token", http.StatusUnauthorized)
		return
	}

	file, err := os.Open("dataset.xml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	readAll, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := XmlData{}
	xml.Unmarshal(readAll, &data)

	result := make([]User, 0, 50)

	q := r.URL.Query()
	query := q.Get("query")
	for _, row := range data.Rows {
		if query != "" {
			if !(strings.Contains(row.FirstName, query) ||
				strings.Contains(row.LastName, query) ||
				strings.Contains(row.About, query)) {
				continue
			}
		}

		result = append(result, User{
			Id:     row.Id,
			Name:   row.FirstName + " " + row.LastName,
			Age:    row.Age,
			About:  row.About,
			Gender: row.Gender,
		})
	}

	orderBy, err := strconv.Atoi(q.Get("order_by"))
	if err != nil {
		http.Error(w, "ErrorBadOrderField", http.StatusBadRequest)
		return
	}

	if !(orderBy == OrderByAsc || orderBy == OrderByDesc || orderBy == OrderByAsIs) {
		sendJsonError(w, "ErrorBadOrderField", http.StatusBadRequest)
		return
	}

	if orderBy != OrderByAsIs {
		var sortFunc func(u1, u2 User) bool

		switch q.Get("order_field") {
		case "Id":
			sortFunc = func(u1, u2 User) bool {
				return u1.Id < u2.Id
			}
		case "Age":
			sortFunc = func(u1, u2 User) bool {
				return u1.Age < u2.Age
			}
		case "Name":
			fallthrough
		case "":
			sortFunc = func(u1, u2 User) bool {
				return u1.Name < u2.Name
			}
		default:
			sendJsonError(w, "ErrorBadOrderField", http.StatusBadRequest)
			return
		}

		if orderBy < 0 {
			sort.Slice(result, func(i, j int) bool {
				return sortFunc(result[i], result[j])
			})
		} else {
			sort.Slice(result, func(i, j int) bool {
				return !sortFunc(result[i], result[j])
			})
		}
	}

	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	offset, err := strconv.Atoi(q.Get("offset"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if limit > 0 {
		from := offset
		if from > len(result)-1 {
			result = []User{}
		} else {
			to := offset + limit
			if to > len(result) {
				to = len(result)
			}

			result = result[from:to]
		}
	}

	resultJson, err := json.Marshal(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resultJson)
}

func sendJsonError(w http.ResponseWriter, error string, code int) {
	resultJson, err := json.Marshal(SearchErrorResponse{error})
	if err != nil {
		http.Error(w, "Fail create json with error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(resultJson)
}

func TestSearchClient_FindUsers(t *testing.T) {
	cases := []testCase{
		{
			request: &SearchRequest{
				Limit: -1,
			},
			result: nil,
			errMsg: "limit must be > 0",
		},
		{
			request: &SearchRequest{
				Offset: -1,
			},
			result: nil,
			errMsg: "offset must be > 0",
		},
		{
			request: &SearchRequest{
				OrderBy: 5,
			},
			result: nil,
			errMsg: "OrderFeld",
		},
		{
			request: &SearchRequest{
				Limit: 1,
				Query: "Boyd",
			},

			result: &SearchResponse{
				Users: []User{
					{
						Id:     0,
						Name:   "Boyd Wolf",
						Age:    22,
						About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
						Gender: "male",
					},
				},
				NextPage: false,
			},
			errMsg: "",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	for caseNum, item := range cases {
		if item.errMsg == "" && item.result == nil {
			t.Errorf("[%d] bad case, booth are nil", caseNum)
			continue
		} else if item.errMsg != "" && item.result != nil {
			t.Errorf("[%d] bad case, booth are not nil", caseNum)
			continue
		}

		client := SearchClient{AccessToken, server.URL}
		users, err := client.FindUsers(*item.request)

		if err == nil && users == nil {
			t.Errorf("[%d] bad result error and users nil", caseNum)
			continue
		} else if err != nil && users != nil {
			t.Errorf("[%d] bad result error and users not nil", caseNum)
			continue
		}

		if err != nil && item.errMsg == "" {
			t.Errorf("[%d] expected result, actual got error: %#v", caseNum, err)
			continue
		} else if err == nil && item.errMsg != "" {
			t.Errorf("[%d] expected error, actual got result: %#v", caseNum, users)
			continue
		}

		if err == nil {
			if !reflect.DeepEqual(users, item.result) {
				t.Errorf("[%d] expected equals result", caseNum)
			}
		} else if users == nil {
			if !strings.Contains(err.Error(), item.errMsg) {
				t.Errorf("[%d] expected contains errMsg: %s actual: %s", caseNum, item.errMsg, err.Error())
			}
		} else {
			t.Errorf("[%d] unexpected state", caseNum)
		}
	}
}

func TestSearchClient_BadToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	client := SearchClient{"erf", server.URL}
	users, err := client.FindUsers(SearchRequest{
		Limit: 26,
	})

	if users != nil {
		t.Errorf("Users not nil")
	}

	if err == nil {
		t.Errorf("err not nil")
		return
	}

	if err.Error() != "Bad AccessToken" {
		t.Errorf("len(users.Users) != 25: %v", len(users.Users))
	}
}

func TestSearchClient_BadJson(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "None")
		if err != nil {
			panic(err)
		}
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("error is nil")
		return
	}
	if !strings.HasPrefix(err.Error(), "cant unpack result json: ") {
		t.Errorf("Invalid error: %v", err)
	}
}

func TestSearchClient_InternalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fatal error", http.StatusInternalServerError)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("error is nil")
		return
	}
	if err.Error() != "SearchServer fatal error" {
		t.Errorf("Invalid error: %v", err)
	}
}

func TestSearchClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
		return
	}
	if !strings.HasPrefix(err.Error(), "timeout for") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestSearchClient_ServerNotExist(t *testing.T) {
	client := SearchClient{AccessToken, ""}

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
		return
	}
	if !strings.HasPrefix(err.Error(), "unknown error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestSearchClient_OverLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	client := SearchClient{AccessToken, server.URL}
	users, err := client.FindUsers(SearchRequest{
		Limit: 26,
	})

	if err != nil {
		t.Errorf("err not nil: %v", err)
	}

	if !users.NextPage {
		t.Errorf("next page not true")
	}

	if len(users.Users) != 25 {
		t.Errorf("len(users.Users) != 25: %v", len(users.Users))
	}
}

func TestCantUnpackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Some Error", http.StatusBadRequest)
	}))
	defer server.Close()

	client := SearchClient{AccessToken, server.URL}
	users, err := client.FindUsers(SearchRequest{})

	if users != nil {
		t.Errorf("users not nil")
	}

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownBadRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendJsonError(w, "Unknown Error", http.StatusBadRequest)
	}))
	defer server.Close()
	client := SearchClient{AccessToken, server.URL}

	users, err := client.FindUsers(SearchRequest{})

	if users != nil {
		t.Errorf("users not nil")
	}

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}
