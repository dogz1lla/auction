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
  - [ ] create a border around the 'create auction' element on the admin page;
  - [ ] make the list of auctions an html table;
  - [ ] upon going to the /home or /admin page create a websocket connection that updates the table
    when there are new highest bidders or the auction ends (expires);

Next:
*/
package main

import (
	"log"
	"net/http"
	"time"

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

	e.GET("/admin", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin-page", roomManager)
	})

	e.GET("/home", func(c echo.Context) error {
		return c.Render(http.StatusOK, "home-page", roomManager)
	})

	e.POST("/create_auction", func(c echo.Context) error {
		closesAtStr := c.FormValue("ClosesAt")
		c.Logger().Printf("GOT TIME STR: %s", closesAtStr)
		closesAt, err := time.Parse(timefmt, closesAtStr)
		if err != nil {
			log.Printf("time parsing error: fmt=%s, to parse=%s, err: %s", timefmt, closesAtStr, err.Error())
		}
		roomManager.CreateAuction(closesAt)
		return c.NoContent(http.StatusOK)
	})

	e.GET("/login", func(c echo.Context) error {
		return c.Render(http.StatusOK, "login-page", mockLoginPage)
	})

	e.POST("/login", func(c echo.Context) error {
		userName := c.FormValue("login")
		user := users.NewUser(userName)
		allUsers = append(allUsers, user)
		if userName == "admin" {
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
		// c.Logger().Print("Ws connection request")
		userName := c.QueryParam("userName")
		roomId := c.QueryParam("roomId")

		room.ServerWs(hub, roomManager, c, userName, roomId)
		return nil
	})

	e.Logger.Fatal(e.Start(":3000"))
}
