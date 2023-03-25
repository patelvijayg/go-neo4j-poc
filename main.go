package main

import (
	"encoding/json"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"io"
	"log"
	"net/http"
	"os"
)

type Neo4jConfiguration struct {
	Url      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

func (nc *Neo4jConfiguration) newDriver() (neo4j.Driver, error) {
	return neo4j.NewDriver(nc.Url, neo4j.BasicAuth(nc.Username, nc.Password, ""))
}

type Employee struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	Email     string `json:"email"`
}

func main() {
	configuration, err := parseConfiguration()
	if err != nil {
		log.Fatal(err)
	}
	driver, err := configuration.newDriver()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection happened")
	defer unsafeClose(driver)
	serveMux := http.NewServeMux()
	//serveMux.HandleFunc("/", defaultHandler)
	serveMux.HandleFunc("/", addHandlerFunc(driver, configuration.Database))
	//serveMux.HandleFunc("/search", searchHandlerFunc(driver, configuration.Database))

	var port string
	var found bool
	if port, found = os.LookupEnv("PORT"); !found {
		port = "8080"
	}
	fmt.Printf("Running on port %s, database is at %s\n", port, configuration.Url)
	panic(http.ListenAndServe(":"+port, serveMux))
}
func parseConfiguration() (*Neo4jConfiguration, error) {

	file, _ := os.Open("conf.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration := Neo4jConfiguration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}
	return &configuration, nil
}
func unsafeClose(closeable io.Closer) {
	if err := closeable.Close(); err != nil {
		log.Fatal(fmt.Errorf("could not close resource: %w", err))
	}
}
func addHandlerFunc(driver neo4j.Driver, database string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var employee Employee
		_ = json.NewDecoder(req.Body).Decode(&employee)
		session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite, DatabaseName: database})
		query := "CREATE (e:Employee {id: $id, firstName: $firstName, Email: $email}) return e"
		var employeeMap map[string]interface{}
		employeeJSON, _ := json.Marshal(employee)
		json.Unmarshal(employeeJSON, &employeeMap)

		result, err := session.Run(query, employeeMap)
		if err != nil {
			return
		}
		if result.Next() {
			fmt.Println(result.Record().Values)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(result.Record().Values)
		}
	}
}
