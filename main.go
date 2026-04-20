package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

func geositeRuleToSrRule(rule *routercommon.Domain) string {
	var srStrategy string

	switch rule.Type {
	case routercommon.Domain_Plain:
		srStrategy = "DOMAIN-KEYWORD"
	case routercommon.Domain_Full:
		srStrategy = "DOMAIN"
	case routercommon.Domain_RootDomain:
		srStrategy = "DOMAIN-SUFFIX"
	case routercommon.Domain_Regex:
		srStrategy = "URL-REGEX"
	default:
		srStrategy = "DOMAIN"
	}
	return srStrategy + "," + rule.Value
}

func geoipRuleToSrRule(rule *routercommon.CIDR) string {
	return "IP-CIDR," + net.IP(rule.Ip).String() + "/" + fmt.Sprintf("%d", rule.Prefix)
}

func processGeositeCategory(category *routercommon.GeoSite) {
	fileName := fmt.Sprintf("%v.list", strings.ToLower(category.CountryCode))
	file, err := os.Create(path.Join("output", "geosite", fileName))
	if err != nil {
		fmt.Printf("failed create file for category %v: %v", category.CountryCode, err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, rule := range category.Domain {
		_, err := writer.Write([]byte(geositeRuleToSrRule(rule) + "\n"))
		if err != nil {
			fmt.Printf("write err: %v\n", err)
			return
		}
	}
	err = writer.Flush()
	if err != nil {
		fmt.Printf("flush err: %v\n", err)
	}
}

func processGeoipCategory(category *routercommon.GeoIP) {
	fileName := fmt.Sprintf("%v.list", strings.ToLower(category.CountryCode))
	file, err := os.Create(path.Join("output", "geoip", fileName))
	if err != nil {
		fmt.Printf("failed create file for category %v: %v", category.CountryCode, err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, rule := range category.Cidr {
		_, err := writer.Write([]byte(geoipRuleToSrRule(rule) + "\n"))
		if err != nil {
			fmt.Printf("write err: %v\n", err)
			return
		}
	}
	err = writer.Flush()
	if err != nil {
		fmt.Printf("flush err: %v\n", err)
	}
}

const WORKERS_COUNT = 8

func processGeosite() {
	// resp, err := http.Get("https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat")
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()

	// fmt.Println("downloading geosite...")
	// bodyBytes, err := io.ReadAll(resp.Body)
	// fmt.Printf("read %d bytes (%.2fM)\n", len(bodyBytes), float64(len(bodyBytes))/1024/1024

	bodyBytes, err := os.ReadFile("temp/geosite.dat")
	if err != nil {
		panic(err)
	}

	fmt.Println("parsing geosite...")
	geositeList := &routercommon.GeoSiteList{}
	err = proto.Unmarshal(bodyBytes, geositeList)
	if err != nil {
		panic(err)
	}
	fmt.Println("parsed geosite")

	err = os.MkdirAll(path.Join("output", "geosite"), 0700)
	wg := sync.WaitGroup{}
	t0 := time.Now()

	jobs := make(chan *routercommon.GeoSite, 128) // test limited channel size

	for i := 0; i < WORKERS_COUNT; i++ {
		wg.Add(1)
		go func(jobs chan *routercommon.GeoSite) {
			defer wg.Done()
			for s := range jobs {
				processGeositeCategory(s)
			}
		}(jobs)
	}

	for _, entry := range geositeList.GetEntry() {
		jobs <- entry
	}

	close(jobs)

	wg.Wait()
	fmt.Printf("finished in %vms\n", time.Since(t0).Milliseconds())
}

func processGeoip() {
	// resp, err := http.Get("https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat")
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()

	// fmt.Println("downloading geoip...")
	// bodyBytes, err := io.ReadAll(resp.Body)
	// fmt.Printf("read %d bytes (%.2fM)\n", len(bodyBytes), float64(len(bodyBytes))/1024/1024

	bodyBytes, err := os.ReadFile("temp/geoip.dat")
	if err != nil {
		panic(err)
	}

	fmt.Println("parsing geoip...")
	geoipList := &routercommon.GeoIPList{}
	err = proto.Unmarshal(bodyBytes, geoipList)
	if err != nil {
		panic(err)
	}
	fmt.Println("parsed geoip")

	err = os.MkdirAll(path.Join("output", "geoip"), 0700)
	wg := sync.WaitGroup{}
	t0 := time.Now()

	jobs := make(chan *routercommon.GeoIP, 128) // test limited channel size

	for range WORKERS_COUNT {
		wg.Add(1)
		go func(jobs chan *routercommon.GeoIP) {
			defer wg.Done()
			for s := range jobs {
				processGeoipCategory(s)
			}
		}(jobs)
	}

	for _, entry := range geoipList.GetEntry() {
		jobs <- entry
	}

	close(jobs)

	wg.Wait()
	fmt.Printf("finished in %vms\n", time.Since(t0).Milliseconds())
}

func main() {
	processGeosite()
	processGeoip()
}
