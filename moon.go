/*
Rise and set times for Sun and Moon.

Copyright 2015 James McHugh

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"github.com/julienschmidt/httprouter"
	"riseset"
	"net/http"
	"os"
	"strconv"
	"time"

  
)

func main() {
	router := httprouter.New()
	router.GET("/", siteroot)
	router.GET("/about", about)
	router.GET("/gettimes", gettimes)
	router.GET("/calendar", calendar)
	router.GET("/paths", paths)
	
	path,_:=os.Getwd()
	fmt.Println("Path:"+path)
	
	router.ServeFiles("/static/*filepath", http.Dir(path+"/static"))
	// router.GET("/hello/:name", Hello)
	var port string
	
	if os.Getenv("PORT") == "" {
		port="8080"
		}else	{
		port=os.Getenv("PORT")
	}
	fmt.Println("Port:"+port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func about(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	myrise := "ignore"
	filename := "about.html" // URL.Path + ".html"
	//fmt.Println(filename)
	t, _ := template.ParseFiles(filename)
	t.Execute(w, myrise)
}

func calendar(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	zon, _ := strconv.ParseFloat(r.URL.Query().Get("zon"), 64)

	type gridrow struct {
		Date string
		Moon riseset.RiseSet
		Sun  riseset.RiseSet
	}
	var arow gridrow
	var Rows []gridrow
	var newdate time.Time

	for i := 0; i < 10; i++ {

		newdate = time.Now().AddDate(0, 0, i)
		//y, m, d := newdate.Date()

		//arow=new(gridrow)
		arow.Date = newdate.Format("02-01-2006")
		arow.Moon = riseset.Riseset(1, newdate, lon, lat, zon)
		arow.Sun = riseset.Riseset(2, newdate, lon, lat, zon)

		Rows = append(Rows, arow)
	}
	t, _ := template.ParseFiles("calendar.html")
	t.Execute(w, &Rows)
}

func siteroot(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body, _ := ioutil.ReadFile("index.html")
	fmt.Fprint(w, string(body))
}

func gettimes(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	a := r.URL.Query().Get("lon")
	b := r.URL.Query().Get("lat")
	c := r.URL.Query().Get("zon")
	var mydata riseset.RiseSet

	if a == "" || b == "" || c == "" {
		mydata = riseset.RiseSet{Rise: "error", Set: "error"}
	} else {
		lon, _ := strconv.ParseFloat(a, 64)
		lat, _ := strconv.ParseFloat(b, 64)
		zon, _ := strconv.ParseFloat(c, 64)
		mydata = riseset.Riseset(1, time.Now(), lon, lat, zon)
	}
	
	json.NewEncoder(w).Encode(mydata)
}

func paths(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	root,_:= os.Getwd()		
	fmt.Fprintf(w, "Start Path: "+ root+"\n")
 	//err := filepath.Walk(root, visit())
  	//fmt.Fprintf(w,"filepath.Walk() returned %v\n", err)
	
	
	d, err := os.Open(root)
	if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
	
    defer d.Close()
    
	fi, err := d.Readdir(-1)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    for _, fi := range fi {
        if fi.Mode().IsRegular() {
            fmt.Fprintf(w,fi.Name()+"\n")
        }
    }
}
	
	
	
	

