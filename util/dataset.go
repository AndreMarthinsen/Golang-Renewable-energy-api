package util

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

type YearAndPercentage struct {
	Year       int32
	Percentage float64
}

type CountryDataset struct {
	mutex sync.RWMutex
	data  map[string]Country
}

func (c *CountryDataset) Initialize(path string) error {
	c.data = make(map[string]Country, 0)
	c.mutex.Lock()
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	nr := csv.NewReader(file)
	for {
		record, err := nr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		countryName := record[0]
		cca3 := record[1]
		if len(cca3) == 3 {
			year, err := strconv.Atoi(record[2])
			if err != nil {
				return err
			}
			percentage, err := strconv.ParseFloat(record[3], 32)
			if err != nil {
				return err
			}
			if _, ok := c.data[cca3]; !ok {
				c.data[cca3] = Country{Name: countryName, YearlyPercentages: make(map[int]float64)}
			}

			c.data[cca3].YearlyPercentages[year] = percentage
		}
	}
	// Calculation of averages
	for cca3, data := range c.data {
		var percentage float64
		startYear := 3000
		endYear := 0

		for year, p := range data.YearlyPercentages {
			if year < startYear {
				startYear = year
			}
			if year > endYear {
				endYear = year
			}
			percentage += p
		}
		temp := c.data[cca3]
		temp.AveragePercentage = percentage / float64(len(data.YearlyPercentages))
		temp.StartYear = startYear
		temp.EndYear = endYear
		c.data[cca3] = temp
	}
	c.mutex.Unlock()
	return nil
}

// GetStatisticsRange returns a list of YearAndPercentage from 'year' to 'lastYear'.
func (c *CountryDataset) GetStatisticsRange(country string, year int, lastYear int) []RenewableStatistics {
	c.mutex.RLock() //TODO: Allow many readers? How?
	var years []RenewableStatistics
	for year <= lastYear {
		if percentage, ok := c.data[country].YearlyPercentages[year]; ok {
			years = append(years, RenewableStatistics{
				Name:       c.data[country].Name,
				Isocode:    country,
				Percentage: percentage,
				Year:       year,
			})
		}
		year += 1
	}
	c.mutex.RUnlock()
	return years
}

func (c *CountryDataset) GetFullName(cca3 string) (string, error) {
	c.mutex.RLock()
	entry, ok := c.data[cca3]
	if ok {
		c.mutex.RUnlock()
		return entry.Name, nil
	}
	c.mutex.RUnlock()
	return "", errors.New("no such entry in dataset")
}

func (c *CountryDataset) GetCountryByName(name string) (string, error) {
	c.mutex.RLock()
	for key, val := range c.data {
		if strings.ToUpper(name) == strings.ToUpper(val.Name) {
			return key, nil
		}
	}
	c.mutex.RUnlock()
	return "", errors.New("no such entry in dataset")
}

// HasCountryInRecords returns true if the country has data in the registry, false otherwise
func (c *CountryDataset) HasCountryInRecords(country string) bool {
	c.mutex.RLock()
	_, ok := c.data[country]
	c.mutex.RUnlock()
	return ok
}

// GetHistoricStatistics returns a slice of the average statistics of all countries.
func (c *CountryDataset) GetHistoricStatistics() []RenewableStatistics {
	c.mutex.RLock()
	statistics := make([]RenewableStatistics, 0)
	for cca3, data := range c.data {
		statistics = append(statistics, RenewableStatistics{
			Name:       data.Name,
			Isocode:    cca3,
			Year:       0,
			Percentage: data.AveragePercentage,
		})
	}
	c.mutex.RUnlock()
	return statistics
}

func (c *CountryDataset) GetStatistic(country string) (RenewableStatistics, error) {
	c.mutex.RLock()
	data, ok := c.data[country]
	if ok {
		c.mutex.RUnlock()
		return RenewableStatistics{
			Name:       data.Name,
			Isocode:    country,
			Year:       data.EndYear,
			Percentage: data.YearlyPercentages[data.EndYear],
		}, nil
	}
	c.mutex.RUnlock()
	return RenewableStatistics{}, errors.New("country not on record")
}

// GetStatistics returns a slice with the statistics for the last year on record
// for each country
func (c *CountryDataset) GetStatistics() []RenewableStatistics {
	c.mutex.RLock()
	statistics := make([]RenewableStatistics, 0)
	for cca3, data := range c.data {
		statistics = append(statistics, RenewableStatistics{
			Name:       data.Name,
			Isocode:    cca3,
			Year:       data.EndYear,
			Percentage: data.YearlyPercentages[data.EndYear],
		})
	}
	c.mutex.RUnlock()
	return statistics
}

// GetFirstYear returns the first year a country has registered renewable data
func (c *CountryDataset) GetFirstYear(country string) int {
	c.mutex.RLock()
	data, ok := c.data[country]
	if ok {
		c.mutex.RUnlock()
		return data.StartYear
	}
	c.mutex.RUnlock()
	return 0
}

// GetLastYear returns the last year a country has registered renewable data
func (c *CountryDataset) GetLastYear(country string) int {
	c.mutex.RLock()
	data, ok := c.data[country]
	if ok {
		c.mutex.RUnlock()
		return data.EndYear
	}
	c.mutex.RUnlock()
	return 0
}

// GetAverage returns the average for a given country
func (c *CountryDataset) GetAverage(country string) (error, float64) {
	data, ok := c.data[country]
	if ok {
		return nil, data.AveragePercentage
	}
	return errors.New("year not on record"), 0.0
}

// GetPercentage returns the percentage for a specific year. Returns an error if the year
// cannot be found in the dataset.
func (c *CountryDataset) GetPercentage(country string, year int) (error, float64) {
	c.mutex.RLock()
	if percentage, ok := c.data[country].YearlyPercentages[year]; ok {
		c.mutex.RUnlock()
		return nil, percentage
	}
	c.mutex.RUnlock()
	return errors.New("year not on record"), 0.0
}

func (c *CountryDataset) GetLengthOfDataset() (error, int) {
	if len(c.data) > 0 {
		return nil, len(c.data)
	} else {
		return errors.New("no dataset initialized"), 0
	}
}

// CalculatePercentage calculates percentage for a given span of years for a specific country
func (c *CountryDataset) CalculatePercentage(code string, startYear int, endYear int) (float64, error) {
	c.mutex.RLock()
	if data, ok := c.data[code]; ok {
		var percentage float64
		var yearSpan float64
		if startYear < c.GetFirstYear(code) {
			startYear = c.GetFirstYear(code)
		}
		if endYear > c.GetLastYear(code) {
			endYear = c.GetLastYear(code)
		}
		for i := startYear; i <= endYear; i++ {
			percentage += data.YearlyPercentages[i]
			yearSpan++
		}
		percentage /= yearSpan
		return percentage, nil
	}
	c.mutex.RUnlock()
	return 0.0, errors.New("country not on record")
}
