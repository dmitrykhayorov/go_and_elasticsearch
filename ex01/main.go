package main

import (
	"day03/ex01/db"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type Store interface {
	GetPlaces(limit int, offset int) ([]db.Place, int, error)
}

// InitPagesRef Terrible((
func InitPagesRef(pages *[]db.PageList, currentPage int, totalCount int, numOfElemOnPage int) {

	lastPage := totalCount / numOfElemOnPage
	fmt.Println(currentPage, " and ", lastPage)
	if currentPage > lastPage {
		currentPage = lastPage

	}
	// Prev
	if currentPage > 0 {
		*pages = append(*pages, db.PageList{
			Name: "Previous",
			Ref:  "/?page=" + strconv.Itoa(currentPage-1)})
	}
	// Next

	if currentPage < lastPage {
		*pages = append(*pages, db.PageList{
			Name: "Next",
			Ref:  "/?page=" + strconv.Itoa(currentPage+1)})
		// Last
	}
	*pages = append(*pages, db.PageList{
		Name: "Last",
		Ref:  "/?page=" + strconv.Itoa(lastPage)})
}

func handlePages(w http.ResponseWriter, r *http.Request, page int) {
	numOfElemOnPage := 10
	var places Store
	places = db.SearchRes{}
	result, count, err := places.GetPlaces(numOfElemOnPage, page*numOfElemOnPage)
	if err != nil {
		http.Error(w, "Error occured while searching", http.StatusBadRequest)
		return
	}
	if page*numOfElemOnPage > count || page < 0 {
		fmt.Println("count in error ", count)
		http.Error(w, "400: page is out of bound: "+strconv.Itoa(page), http.StatusBadRequest)
		return
	}
	//fmt.Println(result)
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		log.Fatalln("can't parse template: ", err)
	}
	err = tmpl.ExecuteTemplate(w, "count", count)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	//io.WriteString(w, "This is my website !\n")
	err = tmpl.ExecuteTemplate(w, "places", result)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var pages []db.PageList

	InitPagesRef(&pages, page, count, numOfElemOnPage)
	err = tmpl.ExecuteTemplate(w, "pages", pages)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

}
func Handler(w http.ResponseWriter, r *http.Request) {
	page := 0
	var err error
	myURL, _ := url.Parse(r.RequestURI)
	params, _ := url.ParseQuery(myURL.RawQuery)
	pageStr := params.Get("page")
	if len(pageStr) > 0 {
		page, err = strconv.Atoi(pageStr)
	}
	if err != nil {
		http.Error(w, "400: page is not numeric", http.StatusBadRequest)
		return
	}
	fmt.Println(params)
	handlePages(w, r, page)
}

func ReccomendHandler(w http.ResponseWriter, r *http.Request) {
	myURL, _ := url.Parse(r.RequestURI)
	params, _ := url.ParseQuery(myURL.RawQuery)
	latStr := params.Get("lat")
	lonStr := params.Get("lon")
	for name, values := range r.Header {
		// Loop over all values for the name.
		for _, value := range values {
			fmt.Println(name, value)
		}
	}
	if len(latStr) < 0 && len(lonStr) < 0 {
		http.Error(w, "400: wrong params! "+latStr+" "+lonStr, http.StatusBadRequest)
		return
	}
	_, err := strconv.ParseFloat(latStr, 32)
	if err != nil {
		http.Error(w, "400: wrong params! "+latStr, http.StatusBadRequest)
		return
	}
	_, err = strconv.ParseFloat(latStr, 32)
	if err != nil {
		http.Error(w, "400: wrong params! "+lonStr, http.StatusBadRequest)
		return
	}

	var rec db.Recommendations

	err = rec.GetPlaces(3, 0, latStr, lonStr)
	if err != nil {
		http.Error(w, "error occurred while parsing", http.StatusBadRequest)
	}
	data, err := json.MarshalIndent(rec, "", "    ")
	//fmt.Printf("%s", data)
	if err != nil {
		log.Println("cannot marshal result JSON")
	}

	_, err = w.Write(data)
	if err != nil {
		log.Println("unexpected error")
	}

}

func makeError(faultPage string) []byte {
	preRes := map[string]string{
		"error": "Invalid 'page' value: '" + faultPage + "'",
	}
	res, err := json.Marshal(preRes)
	if err != nil {
		log.Println("cannot create error json")
	}
	return res
}
func setPagesForApi(as *db.ApiSearch, currentPage int, totalCount int) {
	if currentPage == 0 {
		as.PrevPage = 0
	} else {
		as.PrevPage = currentPage - 1
	}
	lastPage := totalCount / 10
	if currentPage < lastPage {
		as.NextPage = currentPage + 1
	} else {
		as.NextPage = lastPage
	}
	as.LastPage = lastPage
}
func ApiJsonHandler(w http.ResponseWriter, r *http.Request) {
	myURL, _ := url.Parse(r.RequestURI)
	params, _ := url.ParseQuery(myURL.RawQuery)
	page := params.Get("page")
	pageNum := 0
	var err error
	if pageNum, err = strconv.Atoi(page); err != nil {
		jErr := makeError(page)
		http.Error(w, string(jErr), http.StatusBadRequest)
	}
	var res db.ApiSearch
	err = res.GetPlaces(10, pageNum*10)
	if err != nil {
		return
	}
	// Make json
	if pageNum*10 > res.Total || pageNum < 0 {
		jErr := makeError(page)
		http.Error(w, string(jErr), http.StatusBadRequest)
	}
	var apiRes db.ApiSearch
	err = apiRes.GetPlaces(10, pageNum*10)
	if err != nil {
		log.Println("error while search: ", err.Error())
	}
	setPagesForApi(&apiRes, pageNum, apiRes.Total)
	data, err := json.MarshalIndent(apiRes, "", "    ")
	if err != nil {
		log.Println("error while marhalling response: ", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		log.Println("error while writing response: ", err.Error())
		return
	}
}

func GenerateJWT() (string, error) {
	secretKey := []byte("thisIsMyKey")
	token := jwt.New(jwt.SigningMethodHS256)

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func TokenGenerationHandler(w http.ResponseWriter, r *http.Request) {
	token, err := GenerateJWT()
	fmt.Println(token)
	if err != nil {
		log.Println("error while generating token: ", err)
		return
	}

	tokenMap := map[string]string{
		"token": token,
	}

	tokenJson, err := json.MarshalIndent(tokenMap, "", "    ")
	if err != nil {
		log.Println("error while wrapping token to json: ", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(tokenJson)
}

func main() {
	http.HandleFunc("/", Handler)
	http.HandleFunc("/api/get_token", TokenGenerationHandler)
	http.HandleFunc("/api/recommend", ReccomendHandler)
	http.HandleFunc("/api/places", ApiJsonHandler)
	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		fmt.Println(err)
	}
}
