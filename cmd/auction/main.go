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
  - [ ] delete views/admin_page.html
  - [x] make the auction list update when an auction is created
  - [x] remove the create_auction endpoint and do that through the ws instead
  - [ ] upon going to the /home or /admin page create a websocket connection that updates the table
    when the auction ends (expires);

Next:
*/
package main

import (
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
	// mockHomePage := templating.MockHomePage()
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
		homePage := room.NewHomePage(true, roomManager)
		//return c.Render(http.StatusOK, "admin-page", roomManager)
		return c.Render(http.StatusOK, "home-page", homePage)
	})

	e.GET("/home", func(c echo.Context) error {
		homePage := room.NewHomePage(false, roomManager)
		return c.Render(http.StatusOK, "home-page", homePage)
	})

	e.GET("/ws_room_updates", func(c echo.Context) error {
		room.ServerRoomUpdatesWs(roomUpdatesHub, roomManager, c)
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
		isAdmin := userName == "admin"
		if isAdmin {
			c.Response().Header().Set("HX-Redirect", "/admin")
		} else {
			c.Response().Header().Set("HX-Redirect", "/home")
		}
		return c.NoContent(http.StatusOK)
	})

	// next two endpoints work in tandem
	e.GET("/auction", func(c echo.Context) error {
		roomId := c.QueryParam("id")
		auctionRoom, err := roomManager.GetRoomById(roomId)
		log.Println(auctionRoom)
		if err != nil {
			c.Logger().Errorf("Room not found: %s", roomId)
			return nil
		}
		auctionPage := room.NewAuctionPage(auctionRoom)
		return c.Render(http.StatusOK, "auction-page", auctionPage)
	})

	e.GET("/ws", func(c echo.Context) error {
		userName := c.QueryParam("userName")
		roomId := c.QueryParam("roomId")

		room.ServerWs(hub, roomManager, c, userName, roomId)
		return nil
	})

	e.Logger.Fatal(e.Start(":3000"))
}
