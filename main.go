package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/tealeg/xlsx"
)

type Prod struct {
	Cprod       string  `xml:"cProd"`
	CEAN        string  `xml:"cEAN"`
	NCM         string  `xml:"NCM,omitempty"`
	DescProduto string  `xml:"xProd,omitempty"`
	Quantidade  float64 `xml:"qCom,omitempty"`
	ValorUni    float64 `xml:"vUnCom"`
}

type Det struct {
	Prod Prod `xml:"prod"`
}

type InfNFe struct {
	Det []Det `xml:"det"`
}

type NFe struct {
	InfNFe InfNFe `xml:"infNFe"`
}

type DataFormat struct {
	NFe NFe `xml:"NFe"`
}

func main() {

	http.HandleFunc("/oi", handler2)
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)

}

func handler2(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile("serv.xlsx")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Write(file)
}

func handler(w http.ResponseWriter, r *http.Request) {

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data := xmlUnMarshal(b)
	f := ConvertXlsx(data)

	bytess := bytes.NewBuffer([]byte{})

	f.Write(bytess)

	b64s := base64.StdEncoding.EncodeToString(bytess.Bytes())
	//	pretty.Println(data)
	//ioutil.WriteFile(header.Filename, b, 0766)

	w.Header().Set("Content-Type", "application/vnd.ms-excel")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(b64s))
	f.Save("serv.xlsx")
}

func xmlUnMarshal(b []byte) DataFormat {

	data := DataFormat{}
	err := xml.Unmarshal(b, &data)
	if nil != err {
		fmt.Println("Error unmarshalling from XML", err)
	}
	return data
}

func ConvertXlsx(d DataFormat) *xlsx.File {

	gtin := make([]string, 0)
	for _, product := range d.NFe.InfNFe.Det {
		if product.Prod.CEAN != "" {
			gtin = append(gtin, product.Prod.CEAN)
		}
	}
	f := xlsx.NewFile()

	var rst map[string]*sqlProduct
	if len(gtin) > 0 {
		connection, err := makeSQL("179.183.30.186", "3215", "produtostbfg", "0ZRtYoqx!|P%@")
		if err != nil {
			log.Println(err)
			return f
		}

		where := " WHERE GTIN IN ('" + strings.Join(gtin, "', '") + "')"
		rst = connection.getCodBarras(where)
		fmt.Println(rst)
	}

	s, err := f.AddSheet("PLAN 1")
	if err != nil {
		log.Println(err)
		return f
	}
	r := s.AddRow()
	r.AddCell().SetString("Codigo_Produto_Nfe")
	r.AddCell().SetString("Codigo_Ean")
	r.AddCell().SetString("Desc_Produto")
	r.AddCell().SetString("NCM")
	r.AddCell().SetString("Quantidade")
	r.AddCell().SetString("Valor_Un")
	r.AddCell().SetString("Codigo_linx")
	r.AddCell().SetString("Desc_Produto_Linx")
	r.AddCell().SetString("Codigo_barras")

	for _, product := range d.NFe.InfNFe.Det {
		r = s.AddRow()
		r.AddCell().SetString(product.Prod.Cprod)
		r.AddCell().SetString(product.Prod.CEAN)
		r.AddCell().SetString(product.Prod.DescProduto)
		r.AddCell().SetString(product.Prod.NCM)
		r.AddCell().SetFloat(product.Prod.Quantidade)
		r.AddCell().SetFloat(product.Prod.ValorUni)
		if sqlitem, ok := rst[product.Prod.CEAN]; ok {
			r.AddCell().SetString(fmt.Sprintf("%s.%s.%02d", sqlitem.Produto, sqlitem.CorProduto, sqlitem.Tamanho))
			r.AddCell().SetString(fmt.Sprintf("%s %s %s", sqlitem.DescProduto, sqlitem.DescColorProd, sqlitem.Grade))
			r.AddCell().SetString(sqlitem.Codbarra)

		}
	}

	//f.Save("server.xlsx")

	return f

}
