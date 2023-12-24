package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	generatedCount   int
	mu               sync.Mutex
	programStartTime time.Time
	config           Config
	proxies          []*url.URL
	client           *http.Client
	fileMutex        sync.Mutex
)

type Config struct {
	Threads int `json:"threads"`
}

func init() {
	rand.Seed(time.Now().UnixNano())

	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	client = &http.Client{}
	initProxies()
}

func initProxies() {
	proxyList, err := ioutil.ReadFile("proxies.txt")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	proxyStrings := strings.Split(string(proxyList), "\n")
	proxies = make([]*url.URL, 0, len(proxyStrings))

	for _, proxyStr := range proxyStrings {
		proxyStr = strings.TrimSpace(proxyStr)
		if proxyStr != "" {
			proxy, err := url.Parse(fmt.Sprintf("http://%s", proxyStr))
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			proxies = append(proxies, proxy)
		}
	}
}

func generate() {
	mu.Lock()
	generatedCount++
	mu.Unlock()

	payload := map[string]string{
		"partnerUserId": fmt.Sprintf("%s-%s-%s-%s-%s", rstr(7), rstr(7), rstr(2), rstr(4), rstr(12)),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return
	}

	proxy := proxies[rand.Intn(len(proxies))]

	req, err := http.NewRequest("POST", "https://api.discord.gx.games/v1/direct-fulfillment", bytes.NewReader(payloadJSON))
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Ch-Ua", "'Opera GX';v='105', 'Chromium';v='119', 'Not?A_Brand';v='24'")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36 OPR/105.0.0.0 (Edition std-1)")

	client.Transport = &http.Transport{Proxy: http.ProxyURL(proxy)}

	resp, err := client.Do(req)
	if err != nil {
		color.Set(color.FgRed)
		defer color.Unset()
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		color.Set(color.FgRed)
		defer color.Unset()
		fmt.Println(err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		if len(body) > 0 {
			var thingy map[string]interface{}
			err := json.Unmarshal(body, &thingy)
			if err != nil {
				color.Set(color.FgRed)
				defer color.Unset()
				fmt.Println(err)
				return
			}
			link := fmt.Sprintf("https://discord.com/billing/partner-promotions/1180231712274387115/%s", thingy["token"])

			color.Set(color.FgGreen)
			defer color.Unset()
			fmt.Println(link)

			fileMutex.Lock()
			defer fileMutex.Unlock()

			file, err := os.OpenFile("promos.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				color.Set(color.FgRed)
				defer color.Unset()
				fmt.Println("Error:", err)
				return
			}
			defer file.Close()

			_, err = file.WriteString(fmt.Sprintf("%s\n", link))
			if err != nil {
				color.Set(color.FgRed)
				defer color.Unset()
				fmt.Println("Error:", err)
				return
			}
		} else {
			fmt.Println("Opera being problematic and not returning promos (happens quite often and it's normal)")
		}
	} else {
		color.Set(color.FgRed)
		defer color.Unset()
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
		fmt.Printf("Response body: %s\n", body)
	}
}

func updateWindowTitle() {
	linksPerHour := float64(generatedCount) / hoursSinceProgramStart()
	title := fmt.Sprintf("Speed: %.2f/h", linksPerHour)
	fmt.Printf("\033]0;%s\007", title)
}

func hoursSinceProgramStart() float64 {
	return time.Since(programStartTime).Hours()
}

func rstr(l int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, l)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func main() {
	programStartTime = time.Now()

	var wg sync.WaitGroup
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				generate()
			}
		}()
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				updateWindowTitle()
			}
		}
	}()
	
	wg.Wait()
}
