package util

import (
	"Assignment2/consts"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCountryDataset_CalculatePercentage(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dataset.CalculatePercentage("NOR", 1965, 1972)
	assert.Nil(t, err)
	_, err = dataset.CalculatePercentage("SWE", 1982, 1972)
	assert.Error(t, err)
	_, err = dataset.CalculatePercentage("NOR", 0, 0)
	assert.Nil(t, err)
	_, err = dataset.CalculatePercentage("INV", 1965, 1972)
	assert.Error(t, err)
}

func TestCountryDataset_Initialize(t *testing.T) {
	var dataset CountryDataset

	assert.Nil(t, dataset.Initialize("."+consts.DataSetPath))
	assert.Error(t, dataset.Initialize("/invalid/path"))
}

func TestCountryDataset_GetAverage(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	err, _ = dataset.GetAverage("NOR")
	assert.Nil(t, err)
	err, _ = dataset.GetAverage("INV")
	assert.Error(t, err)
}

func TestCountryDataset_GetCountryByName(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dataset.GetCountryByName("Norway")
	assert.Nil(t, err)
	_, err = dataset.GetCountryByName("Nroway")
	assert.Error(t, err)
}

func TestCountryDataset_GetFirstYear(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dataset.GetFirstYear("NOR"), 1965)
	assert.Equal(t, dataset.GetFirstYear("NRO"), 0)
}

func TestCountryDataset_GetFullName(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dataset.GetFullName("NOR")
	assert.Nil(t, err)
	_, err = dataset.GetFullName("NRO")
	assert.Error(t, err)
}

func TestCountryDataset_GetHistoricStatistics(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	statistics := make([]RenewableStatistics, 0)
	statistics = dataset.GetHistoricStatistics()
	assert.Equal(t, len(statistics), len(dataset.data))
}

func TestCountryDataset_GetLastYear(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dataset.GetLastYear("NOR"), 2021)
	assert.Equal(t, dataset.GetLastYear("NRO"), 0)
}

func TestCountryDataset_GetLengthOfDataset(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	err, _ = dataset.GetLengthOfDataset()
	assert.Nil(t, err)
}

func TestCountryDataset_GetPercentage(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	err, _ = dataset.GetPercentage("NOR", 1992)
	assert.Nil(t, err)
	err, _ = dataset.GetPercentage("NOR", 2026)
	assert.Error(t, err)
	err, _ = dataset.GetPercentage("NRO", 2005)
	assert.Error(t, err)
	err, _ = dataset.GetPercentage("NRO", 1954)
	assert.Error(t, err)
}

func TestCountryDataset_GetStatistic(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dataset.GetStatistic("NOR")
	assert.Nil(t, err)
	_, err = dataset.GetStatistic("NRO")
	assert.Error(t, err)
}

func TestCountryDataset_GetStatistics(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	correctStatistic := make([]RenewableStatistics, 0)
	correctStatistic = dataset.GetStatistics()
	wrongStatistic := make([]RenewableStatistics, 0)
	assert.NotEqual(t, correctStatistic, wrongStatistic)
}

func TestCountryDataset_GetStatisticRange(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	var dataset2 CountryDataset
	err = dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	dataset.GetStatisticsRange("NOR", 1992, 1993)
	dataset2.GetStatisticsRange("SWE", 1984, 2001)
	assert.NotEqual(t, dataset, dataset2)
}

func TestCountryDataset_HasCountryInRecords(t *testing.T) {
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dataset.HasCountryInRecords("NOR"), true)
	assert.Equal(t, dataset.HasCountryInRecords("NRO"), false)
}
