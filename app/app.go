package app

// Declare what libraries do we need
import (
	"appengine"
	"appengine/datastore"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Each penguin has following fields
type penguin struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Bio           string `json:"bio"`
	VisitCount    int    `json:"visit_count"`
	FishCount     int    `json:"fish_count"`
	BellyrubCount int    `json:"bellyrub_count"`
}

// Array of penguins
var penguins []penguin

// Track time of last read from the DB for caching
var lastUpdateTime time.Time

// Mutex for goroutine safe operations on penguins array
var mutex sync.Mutex

// DB records
type penguinEntity struct {
	Id            string
	VisitCount    int
	FishCount     int
	BellyrubCount int
}

// This function is called when application starts on the server
func init() {
	loadPenguinsJson()
	lastUpdateTime = time.Now().Add(-20 * time.Minute)
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/penguins", penguinsHandler)
	http.HandleFunc("/update", updateHandler)
	http.HandleFunc("/stat/visit", visitHandler)
	http.HandleFunc("/stat/fish", fishHandler)
	http.HandleFunc("/stat/bellyrub", bellyrubHandler)
}

// Read out configuration file, which describes what penguins do we have
func loadPenguinsJson() {
	file, err := os.Open("penguins.json")
	if err != nil {
		log.Fatal("Can't read penguins.json:", err)
		return
	}

	jsonParser := json.NewDecoder(file)
	err = jsonParser.Decode(&penguins)
	if err != nil {
		log.Fatal("Can't parse penguins.json:", err)
		return
	}
}

// Display a welcome message on app root
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello! This is Penguin Daycare Simulator backend! Number of penguins loaded: ", len(penguins))
}

// Send penguins array to the mobile app with statistics info
func penguinsHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	mutex.Lock()
	defer mutex.Unlock()
	updatePenguinsStatistics(c)
	p, err := json.Marshal(penguins)
	if err != nil {
		c.Errorf("Can't create JSON response: %v", err)
		return
	}
	fmt.Fprint(w, string(p))
}

// Cache management, don't read from the DB until certain amount of time has passed
func updatePenguinsStatistics(c appengine.Context) {
	if time.Since(lastUpdateTime) <= 10*time.Minute {
		return
	}

	lastUpdateTime = time.Now()
	for i, p := range penguins {
		penguin_db := dbGetPenguin(c, p.Id)
		penguins[i].VisitCount = penguin_db.VisitCount
		penguins[i].FishCount = penguin_db.FishCount
		penguins[i].BellyrubCount = penguin_db.BellyrubCount
	}
}

// Force update handler
func updateHandler(w http.ResponseWriter, r *http.Request) {
	lastUpdateTime = time.Now().Add(-20 * time.Minute)
}

// Handle visits event
func visitHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	penguin_id := r.FormValue("id")
	if penguinExists(penguin_id) {
		penguin_db := dbGetPenguin(c, penguin_id)
		penguin_db.VisitCount += 1
		k := datastore.NewKey(c, "Entity", penguin_id, 0, nil)
		_, _ = datastore.Put(c, k, &penguin_db)
	}
}

// Handle fish event
func fishHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	penguin_id := r.FormValue("id")
	if penguinExists(penguin_id) {
		penguin_db := dbGetPenguin(c, penguin_id)
		penguin_db.FishCount += 1
		k := datastore.NewKey(c, "Entity", penguin_id, 0, nil)
		_, _ = datastore.Put(c, k, &penguin_db)
	}
}

// Handle bellyrub event
func bellyrubHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	penguin_id := r.FormValue("id")
	if penguinExists(penguin_id) {
		penguin_db := dbGetPenguin(c, penguin_id)
		penguin_db.BellyrubCount += 1
		k := datastore.NewKey(c, "Entity", penguin_id, 0, nil)
		_, _ = datastore.Put(c, k, &penguin_db)
	}
}

// Reads a record from the DB
func dbGetPenguin(c appengine.Context, id string) penguinEntity {
	var p penguinEntity
	k := datastore.NewKey(c, "Entity", id, 0, nil)
	if err := datastore.Get(c, k, &p); err != nil {
		p.Id = id
	}
	return p
}

// Checks for a valid penguin id
func penguinExists(id string) bool {
	result := false
	for _, p := range penguins {
		if p.Id == id {
			result = true
			break
		}
	}
	return result
}
