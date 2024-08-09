package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Instance represents an EC2 instance type and its pricing details
type Instance struct {
	InstanceType   string  `json:"InstanceType"`
	VCPUS          int     `json:"VCPUS"`
	Memory         string  `json:"Memory"`
	SpotSavingRate string  `json:"SpotSavingRate"`
	SpotPrice      string  `json:"SpotPrice"`
	PricePerVCPU   float64 `json:"PricePerVCPU"`
}

// SpotData represents the entire dataset including all regions and global top 5 deals
type SpotData struct {
	LastUpdated string                `json:"last_updated"`
	Regions     map[string][]Instance `json:"regions"`
	GlobalTop5  []Instance            `json:"global_top_5"`
}

func main() {
	// Fetch and process the spot instance data
	newData := fetchSpotData()

	// Read the existing data file, if it exists
	oldData, _ := readExistingData("docs/spot_data.json")

	// Compare old and new data, update file if there are changes
	if dataChanged(oldData, newData) {
		writeDataToFile(newData, "docs/spot_data.json")
		log.Println("Data updated and saved.")
	} else {
		log.Println("No changes in data.")
	}
}

// fetchSpotData retrieves spot instance data for all regions
func fetchSpotData() SpotData {
	regions := fetchRegions()
	spotData := SpotData{
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		Regions:     make(map[string][]Instance),
	}

	var allInstances []Instance

	// Fetch data for each region
	for _, region := range regions {
		instances := getSpotDeals(region)
		spotData.Regions[region] = instances
		allInstances = append(allInstances, instances...)
	}

	// Sort all instances by PricePerVCPU
	sort.Slice(allInstances, func(i, j int) bool {
		return allInstances[i].PricePerVCPU < allInstances[j].PricePerVCPU
	})

	// Get top 5 global deals
	if len(allInstances) > 5 {
		spotData.GlobalTop5 = allInstances[:5]
	} else {
		spotData.GlobalTop5 = allInstances
	}

	return spotData
}

// getSpotDeals fetches and processes spot deals for a specific region
func getSpotDeals(region string) []Instance {
	url := fmt.Sprintf("https://ec2.shop?region=%s&filter=ebs,cpu>=4,cpu<=32", region)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching data for region %s: %v", region, err)
		return nil
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var response struct {
		Prices []Instance `json:"Prices"`
	}
	json.Unmarshal(body, &response)

	var highSavingsInstances []Instance
	for _, instance := range response.Prices {
		savingsRate, _ := strconv.Atoi(strings.TrimSuffix(instance.SpotSavingRate, "%"))
		if savingsRate > 50 {
			price, _ := strconv.ParseFloat(instance.SpotPrice, 64)
			instance.PricePerVCPU = price / float64(instance.VCPUS)
			highSavingsInstances = append(highSavingsInstances, instance)
		}
	}

	// Sort instances by PricePerVCPU
	sort.Slice(highSavingsInstances, func(i, j int) bool {
		return highSavingsInstances[i].PricePerVCPU < highSavingsInstances[j].PricePerVCPU
	})

	return highSavingsInstances
}

// fetchRegions retrieves the list of AWS regions
func fetchRegions() []string {
	resp, err := http.Get("https://b0.p.awsstatic.com/locations/1.0/aws/current/locations.json")
	if err != nil {
		log.Fatalf("Error fetching regions: %v", err)
	}
	defer resp.Body.Close()

	var regions map[string]struct {
		Type string `json:"type"`
	}
	json.NewDecoder(resp.Body).Decode(&regions)

	var regionCodes []string
	for code, info := range regions {
		if info.Type == "AWS Region" {
			regionCodes = append(regionCodes, code)
		}
	}

	sort.Strings(regionCodes)
	return regionCodes
}

// readExistingData reads the existing spot data file
func readExistingData(filename string) (SpotData, error) {
	var data SpotData
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(file, &data)
	return data, err
}

// writeDataToFile writes the spot data to a JSON file
func writeDataToFile(data SpotData, filename string) {
	file, _ := json.MarshalIndent(data, "", "  ")
	ioutil.WriteFile(filename, file, 0644)
}

// dataChanged checks if there are any differences between old and new data
func dataChanged(old, new SpotData) bool {
	oldData, _ := json.Marshal(old)
	newData, _ := json.Marshal(new)
	return string(oldData) != string(newData)
}
