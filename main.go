/* MIT License
 *
 * Copyright (c) 2022 Kaplar
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// TODO(fusion): Perhaps have these values loaded from a config file?
const (
	// NOTE(fusion): WorldName MUST be the same used in the server's config.lua
	// or the client will fail to connect to the game.
	cfgWorldName     = "Canary"
	cfgGameHost      = "localhost"
	cfgGamePort      = 7172
	cfgWorldLocation = "BRA" // this must be (?) "USA", "EUR", or "BRA"
	cfgWorldType     = "pvp" // this must be "pvp", "no-pvp", or "pvp-enforced"
	cfgDBUser        = "root"
	cfgDBPwd         = "senha123"
	cfgDBHost        = "localhost:3306"
	cfgDBName        = "canary"
	cfgDBUseTLS      = false

	// NOTE(fusion): I have enabled HTTPS support but I haven't been able to test it
	// myself and the reason is because the client seems to only accept certificates
	// signed by a valid certificate authority so using an invalid or self-signed
	// certificate will result in the client error "SSL handshake failed". Now instead
	// of the client sending a "bad certificate" error back, it instantly abort the
	// connection, causing an "EOF" error.
	cfgUseHTTPS     = false
	cfgHTTPSCertPEM = "local/cert.pem"
	cfgHTTPSKeyPEM  = "local/key.pem"
)

var (
	// NOTE(fusion): Handle to the connected database. This should be the only
	// global variable.
	dbHandle *sql.DB
)

type ClientRequest struct {
	RequestType  string `json:"type"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	StayLoggedIn bool   `json:"stayloggedin,omitempty"`
}

type RequestError struct {
	ErrorCode    int64  `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
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
	Location                   string `json:"location"`
	WorldType                  string `json:"pvptype"`
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

func SendEventSchedule(w http.ResponseWriter, eventList []EventInfo, lastUpdate int64) {
	eventSchedule := EventSchedule{eventList, lastUpdate}
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

func HandleBoostedCreatureRequest(w http.ResponseWriter) {
	// TODO(fusion): Do we want to check `boosted_creature`.`date`?
	enabled := true
	raceId := int64(35)
	row := dbHandle.QueryRow("SELECT `raceid` FROM `boosted_creature`")
	if err := row.Scan(&raceId); err != nil {
		fmt.Printf("failed to load boosted creature: %v\n", err)
		enabled = false
	}
	SendBoostedCreature(w, enabled, raceId)
}

func HandleCacheInfoRequest(w http.ResponseWriter) {
	playersOnline := int64(0)
	row := dbHandle.QueryRow("SELECT COUNT(*) FROM `players_online`")
	if err := row.Scan(&playersOnline); err != nil {
		fmt.Printf("failed to count online players: %v\n", err)
	}
	SendCacheInfo(w, playersOnline)
}

func HandleEventScheduleRequest(w http.ResponseWriter) {
	var eventList []EventInfo

	// NOTE(fusion): It seems that the client doesn't display information about
	// the exact time it starts or ends.

	const oneDay = 24 * time.Hour
	today := time.Now().Truncate(oneDay)

	{ // Test Event 1
		startDate := today
		endDate := today.Add(oneDay)
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
		startDate := today.Add(2 * oneDay)
		endDate := today.Add(4 * oneDay)
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
		startDate := today.Add(3 * oneDay)
		endDate := today.Add(5 * oneDay)
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
		startDate := today.Add(-15 * oneDay)
		endDate := today.Add(oneDay)
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

	SendEventSchedule(w, eventList, time.Now().Unix())
}

func TestPassword(pwd string, hashStr string) bool {
	// TODO(fusion): Add support for different hashing schemes.
	pwdHash := sha1.Sum([]byte(pwd))
	testHash, err := hex.DecodeString(hashStr)
	if err != nil {
		return false
	}

	return bytes.Compare(pwdHash[:], testHash) == 0
}

func GetVocationName(id int64) string {
	switch id {
	default:
	case 0:
		return "None"

	case 1:
		return "Sorcerer"
	case 2:
		return "Druid"
	case 3:
		return "Paladin"
	case 4:
		return "Knight"

	case 5:
		return "Master Sorcerer"
	case 6:
		return "Elder Druid"
	case 7:
		return "Royal Paladin"
	case 8:
		return "Elite Knight"
	}

	// NOTE(fusion): Without this we get a "missing return" error.
	return "None"
}

func GetDailyRewardState(pendingReward bool) int64 {
	if pendingReward {
		return 1
	} else {
		return 0
	}
}

func HandleLoginRequest(w http.ResponseWriter, r *ClientRequest) {
	var (
		err         error
		accId       int64
		accPwdHash  string
		accPremDays int64
	)

	accRow := dbHandle.QueryRow(
		"SELECT `id`, `password`, `premdays`"+
			" FROM `accounts` WHERE `email` = ?", r.Email)
	err = accRow.Scan(&accId, &accPwdHash, &accPremDays)
	if err != nil || !TestPassword(r.Password, accPwdHash) {
		SendRequestError(w, 3, "Invalid email or password.")
		return
	}

	chRows, err := dbHandle.Query(
		"SELECT `name`, `level`, `sex`, `vocation`, `looktype`,"+
			" `lookhead`, `lookbody`, `looklegs`, `lookfeet`, `lookaddons`,"+
			" `lastlogin`, `isreward`, `istutorial`"+
			" FROM `players` WHERE `account_id` = ?", accId)
	if err != nil {
		SendRequestError(w, 1, "Internal error.")
		return
	}
	defer chRows.Close()

	accLastLogin := int64(0)
	characters := make([]CharacterInfo, 0)
	for chRows.Next() {
		var (
			chName          string
			chLevel         int64
			chSex           int64
			chVocId         int64
			chLookType      int64
			chLookHead      int64
			chLookBody      int64
			chLookLegs      int64
			chLookFeet      int64
			chLookAddons    int64
			chLastLogin     int64
			chPendingReward bool
			chIsTutorial    bool
		)

		err = chRows.Scan(&chName, &chLevel, &chSex, &chVocId, &chLookType,
			&chLookHead, &chLookBody, &chLookLegs, &chLookFeet, &chLookAddons,
			&chLastLogin, &chPendingReward, &chIsTutorial)
		if err != nil {
			SendRequestError(w, 1, "Internal error.")
			return
		}

		// NOTE(fusion): Keep the most recent "lastlogin".
		if chLastLogin > accLastLogin {
			accLastLogin = chLastLogin
		}

		characters = append(characters, CharacterInfo{
			WorldId:          0,
			Name:             chName,
			Level:            chLevel,
			Vocation:         GetVocationName(chVocId),
			LookType:         chLookType,
			LookHead:         chLookHead,
			LookBody:         chLookBody,
			LookLegs:         chLookLegs,
			LookFeet:         chLookFeet,
			LookAddons:       chLookAddons,
			DailyRewardState: GetDailyRewardState(chPendingReward),
			IsMale:           chSex == 1,
			Tutorial:         chIsTutorial,
			IsHidden:         false,
			IsMainCharacter:  false,
		})
	}

	accPremEnd := int64(0)
	if accPremDays > 0 {
		const oneDay = 24 * time.Hour
		tmp := time.Now()
		tmp.Truncate(oneDay)
		tmp.Add(time.Duration(accPremDays) * oneDay)
		accPremEnd = tmp.Unix()
	}

	// TODO(fusion): This session key looks weird enough. Does the client
	// expect this or does it expect an UID?
	session := SessionInfo{
		SessionKey:            r.Email + "\n" + r.Password,
		Status:                "active", // TODO(fusion)
		LastLogin:             accLastLogin,
		PremiumEnd:            accPremEnd,
		IsPremium:             accPremEnd > 0,
		ReturningPlayer:       false,
		ReturningNotification: false,
		ShowRewardNews:        false,
		FpsTracking:           false,
		OptionTracking:        false,
	}

	world := WorldInfo{
		Id:                         0,
		Name:                       cfgWorldName,
		ExternalAddress:            cfgGameHost,
		ExternalPort:               cfgGamePort,
		ExternalAddressProtected:   cfgGameHost,
		ExternalPortProtected:      cfgGamePort,
		ExternalAddressUnprotected: cfgGameHost,
		ExternalPortUnprotected:    cfgGamePort,
		Location:                   cfgWorldLocation,
		WorldType:                  cfgWorldType,
		AntiCheatProtection:        false,
		RestrictedStore:            false,
	}

	SendLoginResponse(w, &session, []WorldInfo{world}, characters)
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
		SendRequestError(w, 1, "Ill-formed request.")
		return
	}

	// NOTE(fusion): This is for debugging only.
	//fmt.Printf("%T: %+v\n", clRequest, clRequest)

	switch clRequest.RequestType {
	case "boostedcreature":
		HandleBoostedCreatureRequest(w)
	case "cacheinfo":
		HandleCacheInfoRequest(w)
	case "eventschedule":
		HandleEventScheduleRequest(w)
	case "login":
		HandleLoginRequest(w, &clRequest)
	default:
		SendRequestError(w, 1, "Invalid request.")
	}
}

func main() {
	var (
		err       error
		dbVersion string
	)

	dataSourceName := fmt.Sprintf("%v:%v@tcp(%v)/%v?&tls=%t",
		cfgDBUser, cfgDBPwd, cfgDBHost, cfgDBName, cfgDBUseTLS)

	// NOTE(fusion): We can call dbHandle.Ping instead of querying the version if
	// we are not interested in it to make sure we are connected to the database.
	dbHandle, _ = sql.Open("mysql", dataSourceName)
	err = dbHandle.QueryRow("SELECT VERSION()").Scan(&dbVersion)
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}

	fmt.Printf("Connected to database \"%v\" on \"%v\"\n", cfgDBName, cfgDBHost)
	fmt.Printf("Database version: %v\n", dbVersion)

	// NOTE(fusion): Redirect everything to our handler so we don't have the default
	// "NotFoundHandler" sending "404" messages around.
	http.HandleFunc("/", RequestHandler)

	if cfgUseHTTPS {
		fmt.Printf("Kaplar login server (HTTPS) running...\n")
		err = http.ListenAndServeTLS(":443", cfgHTTPSCertPEM, cfgHTTPSKeyPEM, nil)
	} else {
		fmt.Printf("Kaplar login server (HTTP) running...\n")
		err = http.ListenAndServe(":80", nil)
	}

	if err != nil {
		fmt.Printf("HTTP server failed: %v\n", err)
		return
	}
}
