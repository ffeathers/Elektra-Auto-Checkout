package monitor

import (
	"github.com/obito/cclient"
	utls "github.com/refraction-networking/utls"
)


func checkStock() (bool, bool) {
	acceptheader := "application/vnd.com.amazon.api+json; type=\"cart.add-items/v1\""
	contentheader := "application/vnd.com.amazon.api+json; type=\"cart.add-items.request/v1\""
	
	var data = strings.NewReader(`{"items":[{"asin":"` + productId + `","offerListingId":"` + offerId + `","quantity":1}]}`)
	req, err := http.NewRequest("POST", "https://data.amazon.com/api/marketplaces/ATVPDKIKX0DER/cart/carts/retail/items", data)
	if err != nil {
		log.Fatal(err)
	}
	
	req.Header.Set("x-api-csrf-token", apitoken)
	req.Header.Set("Content-Type", contentheader)
	req.Header.Set("Accept", acceptheader)
	req.Header.Set("User-Agent", "Bestbuy-mApp/202104201730 CFNetwork/1209 Darwin/20.2.0")
	
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	
	if resp.StatusCode == 200 {
		return true, false //In stock, api token refresh is not required
	} else if resp.StatusCode == 404 {
		return false, true //Out of stock, but an api token refresh is required 
	} 
	
	return false, false //Usually status code 422 (out of stock) but an api token refresh is not required
}

func getApiToken(client *http.Client) string {
  	url := "https://www.amazon.com/gp/aw/d/B00M382RJO" //One of many Amazon product pages that contains an embedded api token
  
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	
	apiToken := Parse(string(body), "\"csrfToken\":\"", "\",\"baseAsin\"")
	return apiToken
}

func createSession(client *http.Client) {
	url := "https://www.amazon.com/gp/aws/cart/add-res.html?Quantity.1=1&OfferListingId.1="

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()


	/*body, _ := ioutil.ReadAll(resp.Body)
	sessionid := (Parse(string(body), "ue_sid='", "',\nue_mid='"))
	return string(sessionid)*/
}

func Amazon() {
	client, err := cclient.NewClient(utls.HelloFirefox_Auto, true) //Create an http client with a Firefox TLS fingerprint, set automatic storage of cookies to true
	if err != nil {
		log.Fatal(err)
	}
	
	createSession(&client)
	apiToken := getApiToken(&client)
	
	for {
		inStock, refreshRequired := checkStock(&client, apiToken)
		if inStock {
			return 
		} else {
			if refreshRequired {
				apiToken = getApiToken(&client)
			}
		}
		
		time.Sleep(time.Second * time.Interval())
	}
}
