/*
TODO
  - [x] i should see a list of currently running auctions on the user's home page;
  - [x] use htmx ws to replace the auction info element through the websocket (just use the id);
  - [x] each auction needs to have a "auction room" view, this view needs to have a websocket and a
    websocket manager associated with it;
  - [x] each item on the list should have a button that allows the user to join the auction "room",
    this will connect the user to the websocket;
  - [x] websockets should be able to receive the bid messages; connection msg should contain the
    user's name;
  - [x] add expiration to the rooms; when room expires no new bids are allowed;
  - [x] create a border around the 'create auction' element on the admin page;
  - [x] make the list of auctions an html table (only admin for now);
  - [x] upon going to the /home or /admin page create a websocket connection that updates the table
    when there is new highest bidder;
  - [x] delete views/admin_page.html
  - [x] make the auction list update when an auction is created
  - [x] remove the create_auction endpoint and do that through the ws instead
  - [x] upon going to the /home or /admin page create a websocket connection that updates the table
    when the auction ends (expires);
  - [ ] need to map client ids into actual usernames; map[string]string, uid -> username
*/
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/dogz1lla/auction/internal/room"
	"github.com/dogz1lla/auction/internal/templating"
	"github.com/dogz1lla/auction/internal/users"
)

var (
	upgrader = websocket.Upgrader{}
)

const (
	timefmt = "2006-01-02T15:04"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())

	e.Renderer = templating.NewTemplate()
	mockLoginPage := templating.NewLoginPage()
	var allUsers users.Users

	// TODO make an actual templated page and not just auction room struct

	roomUpdatesHub := room.NewRoomUpdatesHub()
	go roomUpdatesHub.Run()

	hub := room.NewHub(roomUpdatesHub)
	go hub.Run()

	roomManager := room.NewRoomManager()
	go roomManager.Run()

	// next two (three) endpoints work in tandem
	e.GET("/admin", func(c echo.Context) error {
		user, err := allUsers.GetUser("admin")
		if err != nil {
			log.Println(fmt.Sprintf("%v", err))
			return c.NoContent(http.StatusInternalServerError)
		}
		homePage := room.NewHomePage(user, roomManager)
		//return c.Render(http.StatusOK, "admin-page", roomManager)
		return c.Render(http.StatusOK, "home-page", homePage)
	})

	e.GET("/home", func(c echo.Context) error {
		userName := c.QueryParam("userName")
		user, err := allUsers.GetUser(userName)
		if err != nil {
			log.Println(fmt.Sprintf("%v", err))
			return c.NoContent(http.StatusInternalServerError)
		}
		homePage := room.NewHomePage(user, roomManager)
		return c.Render(http.StatusOK, "home-page", homePage)
	})

	e.GET("/ws_room_updates", func(c echo.Context) error {
		userName := c.QueryParam("userName")
		user, err := allUsers.GetUser(userName)
		if err != nil {
			log.Println(fmt.Sprintf("/ws_room_updates: %v", err))
			return c.NoContent(http.StatusInternalServerError)
		}
		room.ServerRoomUpdatesWs(roomUpdatesHub, user, roomManager, c)
		return nil
	})

	// ---
	e.GET("/login", func(c echo.Context) error {
		return c.Render(http.StatusOK, "login-page", mockLoginPage)
	})

	e.POST("/login", func(c echo.Context) error {
		userName := c.FormValue("login")
		user := users.NewUser(userName)
		allUsers = append(allUsers, user)
		// TODO: implement a proper auth
		if userName == "admin" {
			c.Response().Header().Set("HX-Redirect", "/admin")
		} else {
			c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/home?userName=%s", userName))
		}
		return c.NoContent(http.StatusOK)
	})

	// next two endpoints work in tandem
	e.GET("/auction", func(c echo.Context) error {
		roomId := c.QueryParam("id")
		auctionRoom, err := roomManager.GetRoomById(roomId)
		if err != nil {
			c.Logger().Errorf("Room not found: %s", roomId)
			return nil
		}
		userName := c.QueryParam("userName")
		user, err := allUsers.GetUser(userName)
		if err != nil {
			log.Println(fmt.Sprintf("%v", err))
			return c.NoContent(http.StatusInternalServerError)
		}
		auctionPage := room.NewAuctionPage(auctionRoom, user)
		return c.Render(http.StatusOK, "auction-page", auctionPage)
	})

	e.GET("/ws", func(c echo.Context) error {
		userName := c.QueryParam("userName")
		roomId := c.QueryParam("roomId")

		user, err := allUsers.GetUser(userName)
		if err != nil {
			log.Println(fmt.Sprintf("%v", err))
			return c.NoContent(http.StatusBadRequest)
		}
		room.ServerWs(hub, roomManager, c, user, roomId)
		return nil
	})

	e.Logger.Fatal(e.Start(":3000"))
}
