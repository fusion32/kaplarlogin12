package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type RequestError struct {
	ErrorCode    int64  `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type ClientRequest struct {
	RequestType  string `json:"type"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	StayLoggedIn bool   `json:"stayloggedin,omitempty"`
}

type BoostedCreature struct {
	Enabled bool  `json:"boostedcreature"`
	RaceId  int64 `json:"raceid"`
}

type CacheInfo struct {
	PlayersOnline        int64 `json:"playersonline"`
	TwitchStreams        int64 `json:"twitchstreams"`
	TwitchViewers        int64 `json:"twitchviewer"`
	GamingYoutubeStreams int64 `json:"gamingyoutubestreams"`
	GamingYoutubeViewers int64 `json:"gamingyoutubeviewer"`
}

type EventInfo struct {
	Name            string `json:"name"`
	StartDate       int64  `json:"startdate"`
	EndDate         int64  `json:"enddate"`
	SpecialEvent    int64  `json:"specialevent"`
	DisplayPriority int64  `json:"displaypriority"`
	IsSeasonal      bool   `json:"isseasonal"`
	Description     string `json:"description"`
	ColorLight      string `json:"colorlight"`
	ColorDark       string `json:"colordark"`
}

type EventSchedule struct {
	EventList  []EventInfo `json:"eventlist"`
	LastUpdate int64       `json:"lastupdatetimestamp"`
}

// TODO(fusion): I have flagged some of the data sent to the client as "Unused".
// Perhaps there is no problem in not sending it to the client?

type SessionInfo struct {
	SessionKey            string `json:"sessionkey"`
	Status                string `json:"status"` // active, frozen, or suspended
	LastLogin             int64  `json:"lastlogintime"`
	PremiumEnd            int64  `json:"premiumuntil"`
	IsPremium             bool   `json:"ispremium"`
	ReturningPlayer       bool   `json:"isreturner"`
	ReturningNotification bool   `json:"returnernotification"`
	ShowRewardNews        bool   `json:"showrewardnews"`
	FpsTracking           bool   `json:"fpstracking"`
	OptionTracking        bool   `json:"optiontracking"`
	Unused1               bool   `json:"emailcoderequest"`
	Unused2               int64  `json:"tournamentticketpurchasestate"`
}

type WorldInfo struct {
	Id                         int64  `json:"id"`
	Name                       string `json:"name"`
	ExternalAddress            string `json:"externaladdress"`
	ExternalPort               int64  `json:"externalport"`
	ExternalAddressProtected   string `json:"externaladdressprotected"`
	ExternalPortProtected      int64  `json:"externalportprotected"`
	ExternalAddressUnprotected string `json:"externaladdressunprotected"`
	ExternalPortUnprotected    int64  `json:"externalportunprotected"`
	Location                   string `json:"location"` // USA, EUR, BRA
	WorldType                  string `json:"pvptype"`  // pvp, no-pvp, pvp-enforced
	AntiCheatProtection        bool   `json:"anticheatprotection"`
	RestrictedStore            bool   `json:"restrictedstore"`
	Unused1                    bool   `json:"istournamentworld"`
	Unused2                    int64  `json:"previewstate"`
	Unused3                    int64  `json:"currenttournamentphase"`
}

type CharacterInfo struct {
	WorldId          int64  `json:"worldid"`
	Name             string `json:"name"`
	Level            int64  `json:"level"`
	Vocation         string `json:"vocation"`
	LookType         int64  `json:"outfitid"`
	LookHead         int64  `json:"headcolor"`
	LookBody         int64  `json:"torsocolor"`
	LookLegs         int64  `json:"legscolor"`
	LookFeet         int64  `json:"detailcolor"`
	LookAddons       int64  `json:"addonsflags"`
	DailyRewardState int64  `json:"dailyrewardstate"`
	IsMale           bool   `json:"ismale"`
	Tutorial         bool   `json:"tutorial"`
	IsHidden         bool   `json:"ishidden"`
	IsMainCharacter  bool   `json:"ismaincharacter"`
	Unused1          bool   `json:"istournamentparticipant"`
	Unused2          int64  `json:"remainingdailytournamentplaytime"`
}

type GameInfo struct {
	Worlds     []WorldInfo     `json:"worlds"`
	Characters []CharacterInfo `json:"characters"`
}

type LoginResponse struct {
	Session SessionInfo `json:"session"`
	Game    GameInfo    `json:"playdata"`
}

func SendRequestError(w http.ResponseWriter, code int64, message string) {
	requestError := RequestError{code, message}
	encoded, err := json.Marshal(requestError)
	if err != nil {
		panic("failed to JSON encode RequestError struct")
	}
	w.Write(encoded)
}

func SendBoostedCreature(w http.ResponseWriter, enabled bool, raceId int64) {
	boostedCreature := BoostedCreature{enabled, raceId}
	encoded, err := json.Marshal(boostedCreature)
	if err != nil {
		panic("failed to encode BoostedCreature struct")
	}
	w.Write(encoded)
}

func SendCacheInfo(w http.ResponseWriter, playersOnline int64) {
	cacheInfo := CacheInfo{playersOnline, 1, 2, 3, 4}
	encoded, err := json.Marshal(cacheInfo)
	if err != nil {
		panic("failed to encode CacheInfo struct")
	}
	w.Write(encoded)
}

func SendEventSchedule(w http.ResponseWriter, eventList []EventInfo, lastUpdate time.Time) {
	eventSchedule := EventSchedule{eventList, lastUpdate.Unix()}
	encoded, err := json.Marshal(eventSchedule)
	if err != nil {
		panic("failed to encode EventSchedule struct")
	}
	w.Write(encoded)
}

func SendLoginResponse(w http.ResponseWriter, session *SessionInfo, worlds []WorldInfo, characters []CharacterInfo) {
	loginResponse := LoginResponse{
		Session: *session,
		Game: GameInfo{
			Worlds:     worlds,
			Characters: characters,
		},
	}
	encoded, err := json.Marshal(loginResponse)
	if err != nil {
		panic("failed to encode LoginResponse struct")
	}
	w.Write(encoded)
}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/login.php" || r.Method != http.MethodPost {
		return
	}

	var data []byte = make([]byte, r.ContentLength)
	if n, err := r.Body.Read(data); int64(n) != r.ContentLength && err != nil {
		fmt.Printf("failed to read POST data: %v\n", err)
		return
	}

	var clRequest ClientRequest
	if err := json.Unmarshal(data, &clRequest); err != nil {
		fmt.Printf("failed to decode ClientRequest struct: %v", err)
		SendRequestError(w, 3, "Ill-formed request.")
		return
	}

	fmt.Printf("%T: %+v\n", clRequest, clRequest)

	switch clRequest.RequestType {
	case "boostedcreature":
		SendBoostedCreature(w, true, 2)
	case "cacheinfo":
		SendCacheInfo(w, 123)
	case "eventschedule":
		var eventList []EventInfo

		// NOTE(fusion): It seems that the client doesn't display information about
		// the exact time it starts or ends.

		{ // Test Event 1
			startDate := time.Date(2022, 6, 25, 12, 15, 0, 0, time.Local)
			endDate := time.Date(2022, 6, 26, 13, 30, 0, 0, time.Local)
			tmp := EventInfo{
				Name:            "Test Event 1",
				StartDate:       startDate.Unix(),
				EndDate:         endDate.Unix(),
				SpecialEvent:    0,
				DisplayPriority: 1,
				IsSeasonal:      false,
				Description:     "Test Event 1 Description",
				ColorLight:      "#2D7400",
				ColorDark:       "#235C00",
			}
			eventList = append(eventList, tmp)
		}

		{ // Test Event 2
			startDate := time.Date(2022, 6, 27, 6, 0, 0, 0, time.Local)
			endDate := time.Date(2022, 6, 30, 8, 0, 0, 0, time.Local)
			tmp := EventInfo{
				Name:            "Test Event 2",
				StartDate:       startDate.Unix(),
				EndDate:         endDate.Unix(),
				SpecialEvent:    0,
				DisplayPriority: 0,
				IsSeasonal:      false,
				Description:     "Test Event 2 Description",
				ColorLight:      "#2D74FF",
				ColorDark:       "#235CFF",
			}
			eventList = append(eventList, tmp)
		}

		{ // Test Event 3
			startDate := time.Date(2022, 6, 29, 6, 0, 0, 0, time.Local)
			endDate := time.Date(2022, 8, 1, 6, 0, 0, 0, time.Local)
			tmp := EventInfo{
				Name:            "Test Event 3",
				StartDate:       startDate.Unix(),
				EndDate:         endDate.Unix(),
				SpecialEvent:    0,
				DisplayPriority: 0,
				IsSeasonal:      true,
				Description:     "Test Event 3 Description",
				ColorLight:      "#2D74FF",
				ColorDark:       "#235CFF",
			}
			eventList = append(eventList, tmp)
		}

		{ // Test Event 4
			startDate := time.Date(2022, 6, 10, 0, 0, 0, 0, time.Local)
			endDate := time.Date(2022, 6, 26, 0, 0, 0, 0, time.Local)
			tmp := EventInfo{
				Name:            "Test Event 4",
				StartDate:       startDate.Unix(),
				EndDate:         endDate.Unix(),
				SpecialEvent:    0,
				DisplayPriority: 0,
				IsSeasonal:      false,
				Description:     "Test Event 4 Description",
				ColorLight:      "#FF7423",
				ColorDark:       "#FF5C23",
			}
			eventList = append(eventList, tmp)
		}

		SendEventSchedule(w, eventList, time.Now())
	case "login":
		tmpSession := SessionInfo{
			SessionKey:            clRequest.Email + "\n" + clRequest.Password,
			Status:                "active",
			LastLogin:             0,
			PremiumEnd:            0,
			IsPremium:             false,
			ReturningPlayer:       false,
			ReturningNotification: false,
			ShowRewardNews:        false,
			FpsTracking:           false,
			OptionTracking:        false,
		}

		// TODO(fusion): Does this need to be a numeric address?
		commonAddress := "127.0.0.1"
		commonPort := int64(7172)

		tmpWorld := WorldInfo{
			Id:                         0,
			Name:                       "Canary",
			ExternalAddress:            commonAddress,
			ExternalPort:               commonPort,
			ExternalAddressProtected:   commonAddress,
			ExternalPortProtected:      commonPort,
			ExternalAddressUnprotected: commonAddress,
			ExternalPortUnprotected:    commonPort,
			Location:                   "BRA",
			WorldType:                  "pvp",
			AntiCheatProtection:        false,
			RestrictedStore:            false,
		}

		tmpCharacter := CharacterInfo{
			WorldId:          0,
			Name:             "Sorcerer Sample",
			Level:            100,
			Vocation:         "Sorcerer",
			LookType:         35,
			LookHead:         0,
			LookBody:         0,
			LookLegs:         0,
			LookFeet:         0,
			LookAddons:       0,
			DailyRewardState: 0,
			IsMale:           true,
			Tutorial:         false,
			IsHidden:         false,
			IsMainCharacter:  true,
		}

		SendLoginResponse(w, &tmpSession, []WorldInfo{tmpWorld}, []CharacterInfo{tmpCharacter})
	default:
		//SendRequestError(w, 3, "Invalid request type.")
		return
	}
}

// NOTE(fusion): Handle to the connected database. This should be the only global variable.
var dbHandle *sql.DB

func main() {
	var (
		err       error
		dbVersion string
	)

	// TODO(fusion): Have a config file? Have more parameters?
	dbUser := "root"
	dbPwd := "senha123"
	dbHost := "localhost:3306"
	dbName := "canary"
	dbUseTLS := false

	dbDSN := fmt.Sprintf("%v:%v@tcp(%v)/%v?&tls=%t",
		dbUser, dbPwd, dbHost, dbName, dbUseTLS)

	// NOTE(fusion): We can call dbHandle.Ping instead of querying the
	// version if we are not interested in it to make sure we are connected
	// to the database.
	dbHandle, _ = sql.Open("mysql", dbDSN)
	err = dbHandle.QueryRow("SELECT VERSION()").Scan(&dbVersion)
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}

	fmt.Printf("Connected to database \"%v\" on \"%v\"\n", dbName, dbHost)
	fmt.Printf("Database version: %v\n", dbVersion)
	fmt.Printf("Kaplar login server running...\n")

	// NOTE(fusion): Redirect everything to our handler so we don't have the default
	// "NotFoundHandler" sending "404" messages around.
	http.HandleFunc("/", RequestHandler)

	// TODO(fusion): We should be using HTTPS but the Tibia client connection fails
	// with "SSL handshake failed" while we fail with an "EOF" error. After some
	// debugging it seems like the client fails right after we flush the server
	// parameters, certificate, and finished handshake messages. We cannot conclude
	// anything from the errors we get but it can be that the client expects a valid
	// certificate (not self signed)?
	//err = http.ListenAndServeTLS(":443", "local/cert.pem", "local/key.pem", nil)

	err = http.ListenAndServe(":80", nil)
	if err != nil {
		fmt.Printf("HTTP server fail: %v\n", err)
		return
	}
}
