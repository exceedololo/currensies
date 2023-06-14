package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Valutes []Valute `xml:"Valute"`
}

type Valute struct {
	XMLName  xml.Name `xml:"Valute"`
	NumCode  int      `xml:"NumCode"`
	CharCode string   `xml:"CharCode"`
	Nominal  int      `xml:"Nominal"`
	Name     string   `xml:"Name"`
	Value    Float    `xml:"Value"`
}

type CurrencyInfo struct {
	Code       string
	Name       string
	MaxValue   CurrencyValue
	MinValue   CurrencyValue
	TotalValue float32
	Count      int
}

type CurrencyValue struct {
	Value Float
	Date  string
}

type Float float32

func (f *Float) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	d.DecodeElement(&v, &start)
	str := strings.Replace(v, ",", ".", 1)
	if fv, err := strconv.ParseFloat(str, 32); err != nil {
		return err
	} else {
		*f = Float(fv)
		return nil
	}
}

func analyzeCurrencyData(currencyData map[string]CurrencyInfo) {
	for _, currencyInfo := range currencyData {
		averageValue := currencyInfo.TotalValue / float32(currencyInfo.Count)

		fmt.Printf("Валюта: %s, Максимальный курс: %.2f (%s), Минимальный курс: %.2f (%s), Средний курс: %.2f\n",
			currencyInfo.Name, currencyInfo.MaxValue.Value, currencyInfo.MaxValue.Date,
			currencyInfo.MinValue.Value, currencyInfo.MinValue.Date, averageValue)
	}
}

func fetchCurrencyData() map[string]CurrencyInfo {
	client := &http.Client{}

	// Определяем сегодняшнюю дату
	today := time.Now()

	// Предыдущие 90 дней
	previous90Days := make([]time.Time, 0)
	for i := 0; i < 90; i++ {
		date := today.AddDate(0, 0, -i)
		previous90Days = append(previous90Days, date)
	}

	currencyData := make(map[string]CurrencyInfo)

	for _, date := range previous90Days {
		// Форматируем дату в требуемом формате
		dateStr := date.Format("02/01/2006")

		// Формируем URL-адрес запроса с указанием даты
		url := fmt.Sprintf("http://www.cbr.ru/scripts/XML_daily_eng.asp?date_req=%s", dateStr)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Error:", err)
			return currencyData
		}

		// Меняем User-Agent на Firefox
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error:", err)
			return currencyData
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Fatal("can't close response")
			}
		}(resp.Body)

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error:", err)
			return currencyData
		}

		valCurs := ValCurs{}

		decoder := xml.NewDecoder(strings.NewReader(string(body)))
		decoder.CharsetReader = charset.NewReaderLabel

		err = decoder.Decode(&valCurs)
		if err != nil {
			fmt.Println("Error:", err)
			return currencyData
		}

		for _, valute := range valCurs.Valutes {
			currencyInfo, ok := currencyData[valute.CharCode]
			if !ok {
				currencyInfo = CurrencyInfo{
					Code: valute.CharCode,
					Name: valute.Name,
				}
			}

			if float32(valute.Value) > float32(currencyInfo.MaxValue.Value) || currencyInfo.MaxValue.Value == 0.0 {
				currencyInfo.MaxValue.Value = valute.Value
				currencyInfo.MaxValue.Date = dateStr
			}

			if float32(valute.Value) < float32(currencyInfo.MinValue.Value) || currencyInfo.MinValue.Value == 0.0 {
				currencyInfo.MinValue.Value = valute.Value
				currencyInfo.MinValue.Date = dateStr
			}

			currencyInfo.TotalValue += float32(valute.Value)
			currencyInfo.Count++

			currencyData[valute.CharCode] = currencyInfo
		}
	}

	return currencyData
}

func main() {
	currencyData := fetchCurrencyData()

	analyzeCurrencyData(currencyData)
}
