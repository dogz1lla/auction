/*
TODO
  - [x] i should see a list of currently running auctions on the user's home page;
  - [x] use htmx ws to replace the auction info element through the websocket (just use the id);
  - [x] each auction needs to have a "auction room" view, this view needs to have a websocket and a
    websocket manager associated with it;
  - [ ] each item on the list should have a button that allows the user to join the auction "room",
    this will connect the user to the websocket;
  - [x] websockets should be able to receive the bid messages; connection msg should contain the
    user's name;

Next: add expiration to the rooms; when room expires no new bids are allowed;
*/
package main

import (
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

func main() {
	e := echo.New()
	e.Use(middleware.Logger())

	e.Renderer = templating.NewTemplate()
	// mockHomePage := templating.MockHomePage()
	mockLoginPage := templating.NewLoginPage()
	var allUsers users.Users
	// TODO make an actual templated page and not just auction room struct

	hub := room.NewHub()
	go hub.Run()

	roomManager := room.NewRoomManager()
	go roomManager.Run()

	// e.GET("/home", func(c echo.Context) error {
	// 	return c.Render(http.StatusOK, "home-page", mockHomePage)
	// })
	//
	// e.POST("/bid", func(c echo.Context) error {
	// 	id := c.FormValue("id")
	//
	// 	bid, err := strconv.ParseFloat(c.FormValue("bid"), 64)
	// 	if err != nil {
	// 		panic("TODO: validate the bid form value")
	// 	}
	// 	mockHomePage.SetBid(id, bid)
	// 	return c.Render(http.StatusOK, "home-page", mockHomePage)
	// })

	e.GET("/login", func(c echo.Context) error {
		return c.Render(http.StatusOK, "login-page", mockLoginPage)
	})

	e.POST("/login", func(c echo.Context) error {
		userName := c.FormValue("login")
		user := users.NewUser(userName)
		allUsers = append(allUsers, user)
		c.Response().Header().Set("HX-Redirect", "/home")
		return c.NoContent(http.StatusOK)
	})

	// next two endpoints work in tandem
	e.GET("/auction", func(c echo.Context) error {
		mockAuctionPage := room.NewMockAuctionPage(roomManager)
		return c.Render(http.StatusOK, "auction-page", mockAuctionPage)
	})

	e.GET("/ws", func(c echo.Context) error {
		// c.Logger().Print("Ws connection request")
		userName := c.QueryParam("userName")
		roomId := c.QueryParam("roomId")

		room.ServerWs(hub, roomManager, c, userName, roomId)
		return nil
	})

	e.Logger.Fatal(e.Start(":3000"))
}
