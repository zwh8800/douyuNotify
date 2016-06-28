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

func checkIfOnline(currentState bool) (bool, map[string]interface{}) {
	resp, err := http.Get("http://acfunfix.sinaapp.com/mama.php?url=http://www.douyutv.com/156277")
	if err != nil {
		glog.Warning(err)
		return currentState, nil
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Warning(err)
		return currentState, nil
	}
	var jsonContent map[string]interface{}
	if err := json.Unmarshal(respData, &jsonContent); err != nil {
		glog.Warning(err)
		return currentState, nil
	}
	code, ok := jsonContent["code"].(float64)
	if !ok {
		glog.Warning(`jsonContent["code"].(float64) not ok`)
		return currentState, nil
	}

	return int(code) == 200, jsonContent
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
					glog.Warning(err)
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
