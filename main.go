package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
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

func processGeositeCategory(category *routercommon.GeoSite, outdir string) {
	fileName := fmt.Sprintf("%v.list", strings.ToLower(category.CountryCode))
	file, err := os.Create(path.Join(outdir, fileName))
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

func processGeoipCategory(category *routercommon.GeoIP, outdir string) {
	fileName := fmt.Sprintf("%v.list", strings.ToLower(category.CountryCode))
	file, err := os.Create(path.Join(outdir, fileName))
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

func processGeosite(data []byte, outdir string, workers int) {
	fmt.Println("parsing geosite...")
	geositeList := &routercommon.GeoSiteList{}
	err := proto.Unmarshal(data, geositeList)
	if err != nil {
		panic(err)
	}
	fmt.Println("parsed geosite")

	err = os.MkdirAll(outdir, 0755)
	wg := sync.WaitGroup{}
	t0 := time.Now()

	jobs := make(chan *routercommon.GeoSite, 128) // test limited channel size

	for range workers {
		wg.Add(1)
		go func(jobs chan *routercommon.GeoSite) {
			defer wg.Done()
			for s := range jobs {
				processGeositeCategory(s, outdir)
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

func processGeoip(data []byte, outdir string, workers int) {
	fmt.Println("parsing geoip...")
	geoipList := &routercommon.GeoIPList{}
	err := proto.Unmarshal(data, geoipList)
	if err != nil {
		panic(err)
	}
	fmt.Println("parsed geoip")

	err = os.MkdirAll(outdir, 0755)
	wg := sync.WaitGroup{}
	t0 := time.Now()

	jobs := make(chan *routercommon.GeoIP, 128) // test limited channel size

	for range workers {
		wg.Add(1)
		go func(jobs chan *routercommon.GeoIP) {
			defer wg.Done()
			for s := range jobs {
				processGeoipCategory(s, outdir)
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

func readData(url string) ([]byte, error) {
	var res []byte
	var err error

	if strings.HasPrefix(strings.ToLower(url), "http://") || strings.HasPrefix(strings.ToLower(url), "https://") {
		// try to resolve as url
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		res, err = io.ReadAll(resp.Body)
	} else {
		// try to resolve as path
		res, err = os.ReadFile(url)
	}

	if err != nil {
		return nil, err
	}
	fmt.Printf("read %d bytes (%.2fM) from %s\n", len(res), float64(len(res))/1024/1024, url)
	return res, err
}

func main() {
	var workerCount int
	var geositeDir string
	var geoipDir string
	var geositePath string
	var geoipPath string

	flag.IntVar(&workerCount, "workers", 8, "workers count to use")
	flag.StringVar(&geositeDir, "geosite-dir", "dist/geosite", "geosite output directory")
	flag.StringVar(&geoipDir, "geoip-dir", "dist/geoip", "geoip output directory")
	flag.StringVar(&geositePath, "geosite-url", "https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat", "geosite file path/url")
	flag.StringVar(&geoipPath, "geoip-url", "https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat", "geoip file path/url")
	flag.Parse()

	if workerCount <= 0 {
		fmt.Println("worker count must be positive")
		return
	}

	if workerCount > 4096 {
		fmt.Println("you sure you need this many?")
		return
	}

	data, err := readData(geositePath)
	if err == nil {
		processGeosite(data, geositeDir, workerCount)
	} else {
		fmt.Printf("geosite failed: %v\n", err)
	}

	data, err = readData(geoipPath)
	if err == nil {
		processGeoip(data, geoipDir, workerCount)
	} else {
		fmt.Printf("geoip failed: %v\n", err)
	}
}
