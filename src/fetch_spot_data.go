package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Instance represents an EC2 instance type and its pricing details
type Instance struct {
	InstanceType   string `json:"InstanceType"`
	VCPUS          int    `json:"VCPUS"`
	Memory         string `json:"Memory"`
	SpotSavingRate string `json:"SpotSavingRate"`
	SpotPrice      string `json:"SpotPrice"`
}

// Response represents the structure of the EC2 shop API response
type Response struct {
	Prices []Instance `json:"Prices"`
}

// Region represents an AWS region and its details
type Region struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	Continent string `json:"continent"`
}

// GlobalDeal represents a spot instance deal with additional information
type GlobalDeal struct {
	InstanceType string  `json:"instanceType"`
	VCPUS        int     `json:"cpus"`
	Memory       string  `json:"memory"`
	SpotPrice    float64 `json:"price"`
	PricePerVCPU float64 `json:"pricePerVCPU"`
	Region       string  `json:"region"`
}

// SpotData represents the entire dataset of spot instance deals
type SpotData struct {
	LastUpdated string                `json:"last_updated"`
	Regions     map[string][]Instance `json:"regions"`
	GlobalTop5  []GlobalDeal          `json:"global_top_5"`
}

func main() {
	// Fetch spot data and write it to a JSON file
	spotData := fetchSpotData()

	file, err := os.Create("docs/spot_data.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(spotData); err != nil {
		log.Fatal(err)
	}
}

// fetchSpotData retrieves spot instance data for all regions
func fetchSpotData() SpotData {
	regions, err := fetchRegions()
	if err != nil {
		log.Fatalf("Error fetching regions: %v", err)
	}

	var wg sync.WaitGroup
	spotData := SpotData{
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		Regions:     make(map[string][]Instance),
	}
	var globalDeals []GlobalDeal
	var mu sync.Mutex

	// Fetch spot deals for each region concurrently
	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			deals, err := getSpotDeals(r)
			if err != nil {
				log.Printf("Error getting spot deals for region %s: %v", r, err)
				return
			}
			mu.Lock()
			spotData.Regions[r] = deals
			if len(deals) > 0 {
				// Add the best deal from this region to globalDeals
				price, _ := strconv.ParseFloat(deals[0].SpotPrice, 64)
				pricePerVCPU := price / float64(deals[0].VCPUS)
				globalDeals = append(globalDeals, GlobalDeal{
					InstanceType: deals[0].InstanceType,
					VCPUS:        deals[0].VCPUS,
					Memory:       deals[0].Memory,
					SpotPrice:    price,
					PricePerVCPU: pricePerVCPU,
					Region:       r,
				})
			}
			mu.Unlock()
		}(region)
	}

	wg.Wait()

	// Sort global deals by price per vCPU
	sort.Slice(globalDeals, func(i, j int) bool {
		return globalDeals[i].PricePerVCPU < globalDeals[j].PricePerVCPU
	})

	// Select top 5 global deals
	if len(globalDeals) > 5 {
		spotData.GlobalTop5 = globalDeals[:5]
	} else {
		spotData.GlobalTop5 = globalDeals
	}

	return spotData
}

// getSpotDeals fetches spot deals for a specific region
func getSpotDeals(region string) ([]Instance, error) {
	url := fmt.Sprintf("https://ec2.shop?region=%s&filter=ebs,cpu>=4,cpu<=32", region)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	// Filter instances with high savings rate (>50%)
	var highSavingsInstances []Instance
	for _, instance := range response.Prices {
		savingsRate, err := strconv.Atoi(strings.TrimSuffix(instance.SpotSavingRate, "%"))
		if err == nil && savingsRate > 50 {
			highSavingsInstances = append(highSavingsInstances, instance)
		}
	}

	// Sort instances by price per vCPU
	sort.Slice(highSavingsInstances, func(i, j int) bool {
		priceI, _ := strconv.ParseFloat(highSavingsInstances[i].SpotPrice, 64)
		priceJ, _ := strconv.ParseFloat(highSavingsInstances[j].SpotPrice, 64)
		ratioI := priceI / float64(highSavingsInstances[i].VCPUS)
		ratioJ := priceJ / float64(highSavingsInstances[j].VCPUS)
		return ratioI < ratioJ
	})

	return highSavingsInstances, nil
}

// fetchRegions retrieves the list of AWS regions
func fetchRegions() ([]string, error) {
	resp, err := http.Get("https://b0.p.awsstatic.com/locations/1.0/aws/current/locations.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var regions map[string]Region
	err = json.Unmarshal(body, &regions)
	if err != nil {
		return nil, err
	}

	// Extract region codes for AWS Regions
	var regionCodes []string
	for _, region := range regions {
		if region.Type == "AWS Region" {
			regionCodes = append(regionCodes, region.Code)
		}
	}

	// Sort region codes alphabetically
	sort.Strings(regionCodes)

	return regionCodes, nil
}
