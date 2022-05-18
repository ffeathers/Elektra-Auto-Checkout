package monitor

import (
	"fmt"
	"github.com/ffeathers/Elektra-Auto-Checkout/elektra"
	"github.com/google/uuid"
	ua "github.com/wux1an/fake-useragent"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type AmazonMonitor struct {
	Id              string
	UserAgent       string
	Proxy           string
	UseProxy        bool
	PollingInterval int
	Sku             string
	OfferId         string
	LoggingDisabled bool
	Active          bool
}

func (monitor *AmazonMonitor) logMessage(msg string) {
	if !monitor.LoggingDisabled {
		log.Println(fmt.Sprintf("[Task %s] [Amazon] %s", monitor.Id, msg))
	}
}

func (monitor *AmazonMonitor) Cancel() {
	monitor.Active = false
	monitor.logMessage("Task canceled")
	//add exit code
}

func (monitor *AmazonMonitor) AmazonCheckStock(client *http.Client, apiToken string) (bool, bool, bool, error) {
	acceptheader := "application/vnd.com.amazon.api+json; type=\"cart.add-items/v1\""
	contentheader := "application/vnd.com.amazon.api+json; type=\"cart.add-items.request/v1\""

	var data = strings.NewReader(fmt.Sprintf(`{"items":[{"asin":"%s","offerListingId":"%s","quantity":1}]}`, monitor.Sku, monitor.OfferId))
	req, err := http.NewRequest("POST", "https://data.amazon.com/api/marketplaces/ATVPDKIKX0DER/cart/carts/retail/items", data)
	if err != nil {
		return false, false, false, err
	}

	req.Header.Set("x-api-csrf-token", apiToken)
	req.Header.Set("Content-Type", contentheader)
	req.Header.Set("Accept", acceptheader)
	req.Header.Set("User-Agent", monitor.UserAgent)
	req.Header.Set("accept-language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return false, false, false, err
	}

	if resp.StatusCode == 200 {
		return true, false, false, nil //In stock, api token refresh is not required
	} else if resp.StatusCode == 404 {
		return false, true, false, nil //Out of stock, but an api token refresh is required
	} else if resp.StatusCode == 422 { //Usually status code 422 (out of stock) but an api token refresh is not required
		return false, false, false, nil
	}

	return false, false, true, nil
}

func (monitor *AmazonMonitor) GetApiToken(client *http.Client) (string, error) {
	url := "https://www.amazon.com/gp/aw/d/B00M382RJO" //One of many Amazon product pages that contains an embedded api token

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("user-agent", monitor.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	apiToken := elektra.Parse(string(body), "\"csrfToken\":\"", "\",\"baseAsin\"")
	return apiToken, nil
}

func (monitor *AmazonMonitor) CreateSession(client *http.Client) error {
	url := "https://www.amazon.com/gp/aws/cart/add-res.html?Quantity.1=1&OfferListingId.1="

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("user-agent", monitor.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (monitor *AmazonMonitor) AmazonMonitorTask() (bool, error) {
	var inStock, refreshRequired, isBanned bool
	var apiToken string

	monitor.Active = true
	monitor.Id = uuid.New().String()

	client, err := elektra.CreateClient(monitor.Proxy)
	if err != nil {
		return false, err
	}

	if monitor.UserAgent == "" {
		monitor.UserAgent = ua.RandomType(ua.Desktop)
	}

	monitor.logMessage("Getting session")
	if !monitor.Active {return false, nil}
	monitor.CreateSession(client)
	

	monitor.logMessage("Getting API token")
	if !monitor.Active {return false, nil}
	apiToken, err = monitor.GetApiToken(client)
	if err != nil {
		return false, err
	}

	for monitor.Active {
		monitor.logMessage("Checking stock")
		inStock, refreshRequired, isBanned, err = monitor.AmazonCheckStock(client, apiToken)
		if err != nil {
			return isBanned, err
		}
		if inStock {
			return isBanned, nil
		} else {
			if refreshRequired {
				if !monitor.Active {return false, nil}
				apiToken, err = monitor.GetApiToken(client)
				if err != nil {
					return isBanned, err
				}
			}
		}

		time.Sleep(time.Second * time.Duration(monitor.PollingInterval))
	}

	return false, nil
}
