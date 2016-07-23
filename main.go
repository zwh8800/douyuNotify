package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/golang/glog"
)

type ServerChanResponse struct {
	Errno   int    `json:"errno"`
	Errmsg  string `json:"errmsg"`
	Dataset string `json:"dataset"`
}

func pushMsgToServerChan(title, description string) error {
	u, err := url.Parse("http://sc.ftqq.com/SCU499T69d8410ac9785fe794ae25fb93ccbf4855f9304ceefac.send")
	if err != nil {
		return err
	}
	query := u.Query()
	query.Add("text", title)
	query.Add("desp", description)
	u.RawQuery = query.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var respBody ServerChanResponse
	if err := json.Unmarshal(respBodyData, &respBody); err != nil {
		return err
	}
	if respBody.Errno != 0 {
		return errors.New(respBody.Errmsg)
	}

	return nil
}

type DouyuLiveData struct {
	Error int    `json:"error"`
	Msg   string `json:"msg"`
	Data  struct {
		RoomID       string `json:"room_id"`
		TagName      string `json:"tag_name"`
		RoomSrc      string `json:"room_src"`
		RoomName     string `json:"room_name"`
		ShowStatus   string `json:"show_status"`
		Online       int    `json:"online"`
		Nickname     string `json:"nickname"`
		HlsURL       string `json:"hls_url"`
		IsPassPlayer int    `json:"is_pass_player"`
		IsTicket     int    `json:"is_ticket"`
		StoreLink    string `json:"storeLink"`
	} `json:"data"`
}

func checkIfOnline(currentState bool) (bool, interface{}) {
	resp, err := http.Get("http://m.douyu.com/html5/live?roomId=156277")
	if err != nil {
		glog.Warningln(err)
		return currentState, nil
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Warningln(err)
		return currentState, nil
	}

	var douyuLiveData DouyuLiveData
	if err := json.Unmarshal(data, &douyuLiveData); err != nil {
		glog.Warningln(err)
		return currentState, nil
	}

	if douyuLiveData.Error != 0 {
		glog.Warningf("douyuLiveData.error = %d, douyuLiveData.msg = %s\n", douyuLiveData.Error, douyuLiveData.Msg)
		return currentState, nil
	}

	return douyuLiveData.Data.ShowStatus == "1", douyuLiveData
}

func JsonStringify(obj interface{}, indent bool) string {
	if indent {
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return ""
		}
		return string(data)
	} else {
		data, err := json.Marshal(obj)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func mainLoop(stopChannel chan bool) {
	currentState := false
	for {
		if thisState, jsonContent := checkIfOnline(currentState); currentState != thisState {
			if thisState {
				glog.Infoln("66开播啦")
				if err := pushMsgToServerChan("66开播啦", JsonStringify(jsonContent, true)); err != nil {
					glog.Warningln(err)
					continue
				}

			} else {
				glog.Infoln("可惜")
			}
			currentState = thisState
		}
		select {
		case <-stopChannel:
			stopChannel <- true
			return
		case <-time.After(5 * time.Minute):
		}
	}
}

func main() {
	defer glog.Flush()
	flag.Parse()
	glog.Infoln("service start")
	stopChannel := make(chan bool)

	go mainLoop(stopChannel)

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt, os.Kill)
	<-signalChannel
	glog.Infoln("shutdown received")
	stopChannel <- true
	<-stopChannel
	glog.Infoln("graceful shutdown")
}
