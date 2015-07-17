package main

import (
    "fmt"
    "log"
    "net/http"
    "github.com/drone/routes"
    "strconv"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "gopkg.in/gorp.v1"
    "encoding/json"
)

type Person struct {
    Id int64
    Firstname string 
    Lastname string 
}

var dbmap = initDb()


func hello(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello World")
}


func whoami(w http.ResponseWriter, r *http.Request){
    params:=r.URL.Query()
    lastName:=params.Get(":last")
    firstName:=params.Get(":first")
    fmt.Fprintf(w, "you are %s %s", firstName, lastName)
}

// Get handler

func getuser(w http.ResponseWriter, r *http.Request){
    params:=r.URL.Query()
    id:=params.Get(":id")
    fmt.Fprintf(w, "get user: id:%s", id)
/*
    user_id, _ := strconv.ParseInt(id, 0, 64)
    if user_id == 1{
        person :=  &Person{Id:1, Firstname:"Tsugunao", Lastname:"Kobayashi"}
        data, err := json.Marshal(person)
        if err != nil {
            fmt.Println(err)
        }
        fmt.Fprintf(w, "%s\n", string(data))
        //routes.ServeJson(w, &person)        
    } else if user_id == 2 {
        person :=  &Person{Id:1, Firstname:"Yuka", Lastname:"Kobayashi"}
        routes.ServeJson(w, &person)
    } else {
        fmt.Fprintf(w, "Error id %d is not defined", id)
    }
*/
    var user Person
    err := dbmap.SelectOne(&user, "SELECT * FROM user WHERE id=?", id)

    if err == nil {
        user_id, _ := strconv.ParseInt(id,0,64)
        content := &Person{
            Id: user_id,
            Firstname: user.Firstname,
            Lastname: user.Lastname,
        }
        routes.ServeJson(w, content)
    } else {
         fmt.Fprintf(w, "Error id %d is not defined", id)
    }
    //curl --noproxy localhost http://localhost:9090/users/1
}

func getusers(w http.ResponseWriter, r *http.Request){
    fmt.Fprintf(w, "get users")
    type Users []Person

    /*
    var users = Users {
        Person{Id:1, Firstname:"Tsugunao", Lastname:"Kobayashi"},
        Person{Id:2, Firstname:"Yuka", Lastname:"Kobayashi"},
    }
    */

    var users []Person 
    _, err := dbmap.Select(&users, "SELECT * FROM user")
    if err == nil {
        routes.ServeJson(w, &users)
    } else {
        fmt.Fprintf(w, "Error: no users in the table")
    }    
    //routes.ServeJson(w, &users)
    // curl --noproxy localhost http://localhost:9090/users
}


func modifyuser(w http.ResponseWriter, r *http.Request){
    params:=r.URL.Query()
    id:=params.Get(":id")
    fmt.Fprintf(w, "user updated: id:%s", id)

    var user Person
    err := dbmap.SelectOne(&user, "SELECT * FROM user WHERE id=?", id)
    if err == nil {
        decoder := json.NewDecoder(r.Body)
        var p Person
        err := decoder.Decode(&p)
        if err != nil {
            fmt.Fprintf(w, "json decode error")
        }

        user_id, _ := strconv.ParseInt(id, 0, 64)
        content := Person{
            Id: user_id,
            Firstname: p.Firstname,
            Lastname: p.Lastname,
        }
        if content.Firstname != "" && content.Lastname != ""{
            _, err = dbmap.Update(&content)
            if err == nil{
                routes.ServeJson(w, content)
            } else {
                checkErr(err, "Update failed")
            }
        } else {
            fmt.Fprintf(w, "Error: fields are empty")
        }
    } else{
        fmt.Fprintf(w, "Error: user not found")
    }
    // curl --noproxy localhost http://localhost:9090/users/3 -X PUT -d '{ "Firstname": "Tsugu", "Lastname":"kobayashi"}'

}

func deleteuser(w http.ResponseWriter, r *http.Request){
    params:=r.URL.Query()
    id:=params.Get(":id")
    fmt.Fprintf(w, "user deleted: id:%s", id)

    var user Person
    err := dbmap.SelectOne(&user, "SELECT id FROM user WHERE id=?", id)
    if err == nil {
        _, err = dbmap.Delete(&user)
        if err == nil {
            fmt.Fprintf(w, "user id: %d is deleted", id)
        } else {
            checkErr(err, "Delete failed")
        }
    }
    //$ curl --noproxy localhost http://localhost:9090/users/3 -X DELETE
}

func adduser(w http.ResponseWriter, r *http.Request){
    decoder := json.NewDecoder(r.Body)
    var p Person  
    err := decoder.Decode(&p)
    if err != nil {
        fmt.Fprintf(w, "json decode error")
    }
    fmt.Fprintf(w, "id:%s name:%s %s", p.Id, p.Firstname, p.Lastname)
    
    if p.Firstname != "" && p.Lastname != "" {
        if insert, _ := dbmap.Exec(`INSERT INTO user (firstname, lastname) VALUES (?, ?)`,
            p.Firstname, p.Lastname);
        insert != nil {
            user_id, err := insert.LastInsertId()
            if err == nil {
                content := &Person{
                    Id: user_id,
                    Firstname: p.Firstname,
                    Lastname: p.Lastname,
                }
                routes.ServeJson(w, content)
            }
        } else {
            fmt.Fprintf(w, "Error: insert failed")
        }
    }
    //curl --noproxy localhost http://localhost:9090/users/ -X POST -d '{ "Firstname": "Tsugunao", "Lastname" : "kobayashi"}'
}

func initDb() *gorp.DbMap {

    db, err := sql.Open("mysql", "root:rootroot@/myapi")
    checkErr(err, "sql.Open failed")
    dbmap := &gorp.DbMap{Db: db, Dialect:gorp.MySQLDialect{"InnoDB", "UTF8"}}
    dbmap.AddTableWithName(Person{}, "User").SetKeys(true, "Id")
    err = dbmap.CreateTablesIfNotExists()
    checkErr(err, "Create table failed")

    return dbmap
}

func checkErr(err error, msg string) {
    if err != nil {
        log.Fatalln(msg, err)
    }
}


func main() {
    //http.HandleFunc("/", hello)
    mux := routes.New()
    //mux.Get("/:last/:first", whoami)
    mux.Get("/users", getusers)
    mux.Get("/users/:id", getuser)
    mux.Put("/users/:id", modifyuser)
    mux.Del("/users/:id", deleteuser)
    mux.Post("/users/", adduser)
    http.Handle("/", mux)


    err := http.ListenAndServe(":9090", nil)
    if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}




