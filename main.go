package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
	ID     int64  `db:"id, primarykey, autoincrement"`
	Fname  string `db:"first_name,size:50"`
	Lname  string `db:"last_name,size:50"`
	Email  string `db:"email"`
	Ipaddr string `db:"ip_address"`
}

type App struct {
	DbMap *gorp.DbMap
}

func checkErr(w http.ResponseWriter, err error, msg string, fatal bool) {

	if err != nil {
		if w != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			return
		} else {
			if fatal {
				log.Fatalln(msg, err)
				return
			} else {
				log.Printf(msg, err)
				return
			}
		}

	}
}

func (a *App) Init() error {
	db, err := sql.Open("sqlite3", "Clients.db")
	if err != nil {
		return err
	}
	a.DbMap = initDb(db)
	return nil
}

func (a *App) Close() {
	if a.DbMap != nil {
		a.DbMap.Db.Close()
	}
}

func initDb(db *sql.DB) *gorp.DbMap {
	// Use the dbmap to define and create tables
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	// Add a table mapping for the Person struct
	dbmap.AddTableWithName(Client{}, "clients").SetKeys(true, "ID")

	err := dbmap.CreateTablesIfNotExists()
	checkErr(nil, err, "Error creating tables:", true)

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)

	return dbmap
}

func (a *App) ReadClients() []Client {
	// Retrieve all people from the database
	var clients []Client
	_, err := a.DbMap.Select(&clients, "SELECT * FROM clients")
	checkErr(nil, err, "Error reading Clients:", true)

	return clients
}

func (a *App) ReadClient(id int) Client {
	// Retrieve all people from the database
	var client Client
	err := a.DbMap.SelectOne(&client, "SELECT * FROM clients WHERE id=?", id)
	checkErr(nil, err, "Error reading Client:", true)

	return client
}

func (a *App) UpdateClient(client *Client) error {
	_, err := a.DbMap.Update(client)
	checkErr(nil, err, "Error updating Client: ", true)

	return err
}

func (a *App) ViewClientsHandler(w http.ResponseWriter, r *http.Request) {

	clientsComponent = showClients(a.ReadClients())
	clientsComponent.Render(context.Background(), w)

}

func (a *App) CreateClientHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method == http.MethodGet {
		addPage().Render(ctx, w)
		// addClientComponent.Render(ctx, w)

		return
	} else {
		r.ParseForm()
		fname := r.FormValue("fname")
		lname := r.FormValue("lname")
		email := r.FormValue("email")
		ipaddr := r.FormValue("ipaddr")

		// Validate the required fields
		if fname == "" || lname == "" || email == "" || ipaddr == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Create a new client
		newClient := Client{
			// ID:     newID, // ID will be handled automatically by the database
			Fname:  fname,
			Lname:  lname,
			Email:  email,
			Ipaddr: ipaddr,
		}

		// Insert the new client into the database
		err := a.DbMap.Insert(&newClient)
		checkErr(w, err, "Error inserting client into the database", true)
		// Redirect or respond as needed
		home().Render(ctx, w)
		//Or
		// addClientComponent = addPage()
		// addClientComponent.Render(ctx, w)
		return

	}
}

func (a *App) UpdateClientHandler(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	vars := mux.Vars(r)
	id := vars["id"]
	idint, err := strconv.Atoi(id)
	checkErr(w, err, "ERROR converting string ID to int", true)
	fmt.Println("〓〓•_•_•◥ ID is:", id, "◤•_•_•〓〓")

	// Fetch the client from the database based on the ID
	var client = Client{}
	err = a.DbMap.SelectOne(&client, "SELECT * FROM clients WHERE id = ?", idint)
	checkErr(w, err, "ERROR SELECTING ID fron the database", true)

	if r.Method == http.MethodGet {
		//RENDER EDITPAGE
		// temp := ViewModelWithCurrentPage{CurrentPage: r.URL.Path, ViewModel: client}
		editClient(client).Render(ctx, w)

		return
	}
	if r.Method == http.MethodPost {

		r.ParseForm()
		client.ID = int64(idint)
		client.Fname = r.FormValue("fname")
		client.Lname = r.FormValue("lname")
		client.Email = r.FormValue("email")
		client.Ipaddr = r.FormValue("ipaddr")

		if client.Fname == "" || client.Lname == "" || client.Email == "" || client.Ipaddr == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Update the client in the database
		err := a.UpdateClient(&client)
		checkErr(w, err, "ERROR CALLING UPDATE CLIENT()", true)

		fmt.Printf("〓〓•_•_•◥ FUCKIN RES is: %+v ◤•_•_•〓〓", &client)

		// http.Redirect(w, r, "/clients", http.StatusSeeOther)
		clientsComponent = showClients(a.ReadClients())
		clientsComponent.Render(context.Background(), w)
		return
	}

}

func (a *App) DeleteClientHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := a.DbMap.Exec("delete from clients where ID=?", id)
	checkErr(w, err, "ERROR DELETING ID fron the database", true)
	http.Redirect(w, r, "/clients", http.StatusOK)

	// clientsComponent = showClients(a.ReadClients())
	// clientsComponent.Render(context.Background(), w)

}

type SearchOptions struct {
	ID     *int64
	Fname  string
	Lname  string
	Email  string
	Ipaddr string
}

func (a *App) SearchClients(searchOptions SearchOptions) ([]Client, error) {
	// Start building the SQL query
	query := "SELECT * FROM clients WHERE 1=1"

	if searchOptions.ID != nil {
		query += fmt.Sprintf(" AND id = %d", *searchOptions.ID)
	}

	if searchOptions.Fname != "" {
		query += fmt.Sprintf(" AND first_name = '%s'", searchOptions.Fname)
	}

	if searchOptions.Lname != "" {
		query += fmt.Sprintf(" AND last_name = '%s'", searchOptions.Lname)
	}

	if searchOptions.Email != "" {
		query += fmt.Sprintf(" AND email = '%s'", searchOptions.Email)
	}

	if searchOptions.Ipaddr != "" {
		query += fmt.Sprintf(" AND ip_address = '%s'", searchOptions.Ipaddr)
	}

	var clients []Client
	_, err := a.DbMap.Select(&clients, query)
	if err != nil {
		return nil, err
	}

	return clients, nil
}

func (a *App) SearchClientHandler(w http.ResponseWriter, r *http.Request) {

	// ctx := r.Context()
	ctx := context.Background()

	if r.Method == http.MethodGet {
		// render the Search template
		searchComponent.Render(ctx, w)

	} else {

		r.ParseForm()
		idStr := r.PostFormValue("id")
		var id *int64
		if idStr != "" {
			idVal, err := strconv.ParseInt(idStr, 10, 64)
			checkErr(w, err, "Error Invalid ID format", false)
			id = &idVal
		}

		options := SearchOptions{
			ID:     id,
			Fname:  r.PostFormValue("fname"),
			Lname:  r.PostFormValue("lname"),
			Email:  r.PostFormValue("email"),
			Ipaddr: r.PostFormValue("ipAddr"),
		}

		queryResult, err := a.SearchClients(options)
		checkErr(w, err, "Error searching clients", false)
		//log test
		// fmt.Printf("〓〓•_•_•◥ QUERYRESULT is: %+v ◤•_•_•〓〓", queryResult)

		// showClients(queryResult).Render(ctx, w)
		for _, clt := range queryResult {
			showClient(clt).Render(ctx, w)
		}
	}
}

func (a *App) HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	ctx := r.Context()
	homeComponent.Render(ctx, w)
	// home().Render(ctx,w) //or
}

func (a *App) SingleClientHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]
	idint, err := strconv.Atoi(id)
	checkErr(w, err, "ERROR CONVERTING CLIENT ID", false)

	clientComponent = showClient(a.ReadClient(idint))
	// templ.Handler(clientComponent)
	clientComponent.Render(context.Background(), w)

}

// fmt.Printf("\n〓〓•_•_•◥ FUCKIN ID is: %+v ◤•_•_•〓〓\n", int64(idint))

func (a *App) SingleClientUpdateHandler(w http.ResponseWriter, r *http.Request) {
	// ctx := r.Context()

	vars := mux.Vars(r)
	id := vars["id"]
	idint, err := strconv.Atoi(id)
	checkErr(w, err, "ERROR converting string ID to int", true)
	fmt.Println("〓〓•_•_•◥ ID is:", id, "◤•_•_•〓〓")

	// Fetch the client from the database based on the ID
	var client = Client{}
	err = a.DbMap.SelectOne(&client, "SELECT * FROM clients WHERE id = ?", idint)
	checkErr(w, err, "ERROR SELECTING ID fron the database", true)

	if r.Method == http.MethodGet {
		editClient(a.ReadClient(idint)).Render(context.Background(), w)
	} else if r.Method == http.MethodPut {
		r.ParseForm()
		client.ID = int64(idint)
		client.Fname = r.FormValue("fname")
		client.Lname = r.FormValue("lname")
		client.Email = r.FormValue("email")
		client.Ipaddr = r.FormValue("ipaddr")

		if client.Fname == "" || client.Lname == "" || client.Email == "" || client.Ipaddr == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Update the client in the database
		err := a.UpdateClient(&client)
		checkErr(w, err, "ERROR CALLING UPDATE CLIENT()", true)

		fmt.Printf("〓〓•_•_•◥ FUCKIN RES is: %+v ◤•_•_•〓〓", &client)

		// http.Redirect(w, r, "/clients", http.StatusSeeOther)
		// clientsComponent = showClients(a.ReadClients())
		// clientsComponent.Render(context.Background(), w)
		showClient(a.ReadClient(idint)).Render(context.Background(), w)
		return
		//maybe get r.URL.Path then render the same path
	}

}

type ViewModelWithCurrentPage struct {
	CurrentPage string
	ViewModel   Client
	//TODO
}

var (
	homeComponent    templ.Component
	clientsComponent templ.Component
	searchComponent  templ.Component

	clientComponent     templ.Component
	editClientComponent templ.Component
	addClientComponent  templ.Component
)

func main() {

	port := flag.String("port", "4000", "HTTP network port for http://localhost:")
	flag.Parse()

	homeComponent = home()
	templ.Handler(homeComponent)

	searchComponent = search()
	templ.Handler(searchComponent)

	clientsComponent = showClients([]Client{})
	templ.Handler(clientsComponent)

	clientComponent = showClient(Client{})
	templ.Handler(clientComponent)

	// editComponent = editPage(ViewModelWithCurrentPage{})
	// templ.Handler(editComponent)

	editClientComponent = editClient(Client{})
	templ.Handler(editClientComponent)

	addClientComponent = addPage()
	templ.Handler(addClientComponent)

	app := &App{
		//
	}
	err := app.Init()
	checkErr(nil, err, "Error initializing the app ", true)
	defer app.Close()

	r := mux.NewRouter().
		SkipClean(true).
		StrictSlash(true)

	fs := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	r.PathPrefix("/static/").Handler(fs).Methods("GET")

	r.HandleFunc("/", app.HomeHandler) //.Methods("GET")
	r.HandleFunc("/clients", app.ViewClientsHandler).Methods("GET")
	r.HandleFunc("/clients/add", app.CreateClientHandler).Methods("GET")
	r.HandleFunc("/clients/add", app.CreateClientHandler).Methods("POST")
	r.HandleFunc("/clients/delete/{id}", app.DeleteClientHandler).Methods("DELETE")
	r.HandleFunc("/clients/update/{id}", app.UpdateClientHandler).Methods("GET")
	r.HandleFunc("/clients/update/{id}", app.UpdateClientHandler).Methods("POST")
	r.HandleFunc("/clients/search", app.SearchClientHandler).Methods("GET")
	r.HandleFunc("/clients/search", app.SearchClientHandler).Methods("POST")

	r.HandleFunc("/client/{id}", app.SingleClientHandler).Methods("GET")
	r.HandleFunc("/client/{id}/edit", app.SingleClientUpdateHandler).Methods("GET")
	r.HandleFunc("/client/{id}/edit", app.SingleClientUpdateHandler).Methods("PUT")

	server := &http.Server{
		Addr:    ":" + *port,
		Handler: r,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Server error: %s\n", err)
		}
	}()

	log.Println("••• Server Listening on •• http://localhost:" + *port)

	stopChan := make(chan os.Signal, 8)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan // wait for SIGINT
	log.Println("•• Shutting down server...")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second)
	server.Shutdown(ctx)
	<-ctx.Done()
	cancel()

	log.Println("• Server gracefully stopped.")
}

// // Disable FileServer Directory Listings
// func neuter(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if strings.HasSuffix(r.URL.Path, "/") {
// 			http.NotFound(w, r)
// 			return
// 		}
// 		next.ServeHTTP(w, r)
// 	})
// }

//sample Middleware ... usage: http.Handle("/", loggingHandler(indexHandler))
// func loggingHandler(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//    start := time.Now()
//    log.Printf("Started %s %s", r.Method, r.URL.Path)
//    next.ServeHTTP(w, r)
//    log.Printf("Completed %s in %v", r.URL.Path, time.Since(start))
// 	})
//    }
