// Copyright (c) 2018, Randy Westlund. All rights reserved.
// This code is under the BSD-2-Clause license.

package quickbooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	null "gopkg.in/guregu/null.v4"
)

// Customer represents a QuickBooks Customer object.
type Customer struct {
	ID                 string          `json:"Id,omitempty"`
	SyncToken          string          `json:",omitempty"`
	MetaData           MetaData        `json:",omitempty"`
	Title              null.String     `json:",omitempty"`
	GivenName          null.String     `json:",omitempty"`
	MiddleName         null.String     `json:",omitempty"`
	FamilyName         null.String     `json:",omitempty"`
	Suffix             null.String     `json:",omitempty"`
	DisplayName        string          `json:",omitempty"`
	FullyQualifiedName null.String     `json:",omitempty"`
	CompanyName        null.String     `json:",omitempty"`
	PrintOnCheckName   string          `json:",omitempty"`
	Active             bool            `json:",omitempty"`
	PrimaryPhone       TelephoneNumber `json:",omitempty"`
	AlternatePhone     TelephoneNumber `json:",omitempty"`
	Mobile             TelephoneNumber `json:",omitempty"`
	Fax                TelephoneNumber `json:",omitempty"`
	PrimaryEmailAddr   *EmailAddress   `json:",omitempty"`
	WebAddr            *WebSiteAddress `json:",omitempty"`
	//DefaultTaxCodeRef
	Taxable              *bool            `json:",omitempty"`
	TaxExemptionReasonID *string          `json:"TaxExemptionReasonId,omitempty"`
	BillAddr             *PhysicalAddress `json:",omitempty"`
	ShipAddr             *PhysicalAddress `json:",omitempty"`
	Notes                string           `json:",omitempty"`
	Job                  null.Bool        `json:",omitempty"`
	BillWithParent       bool             `json:",omitempty"`
	ParentRef            ReferenceType    `json:",omitempty"`
	Level                int              `json:",omitempty"`
	//SalesTermRef
	//PaymentMethodRef
	Balance         json.Number `json:",omitempty"`
	OpenBalanceDate Date        `json:",omitempty"`
	BalanceWithJobs json.Number `json:",omitempty"`
	//CurrencyRef
	IsProject bool `json:"IsProject"`
}

// GetAddress prioritizes the ship address, but falls back on bill address
func (c Customer) GetAddress() PhysicalAddress {
	if c.ShipAddr != nil {
		return *c.ShipAddr
	}
	if c.BillAddr != nil {
		return *c.BillAddr
	}
	return PhysicalAddress{}
}

// GetWebsite de-nests the Website object
func (c Customer) GetWebsite() string {
	if c.WebAddr != nil {
		return c.WebAddr.URI
	}
	return ""
}

// GetPrimaryEmail de-nests the PrimaryEmailAddr object
func (c Customer) GetPrimaryEmail() string {
	if c.PrimaryEmailAddr != nil {
		return c.PrimaryEmailAddr.Address
	}
	return ""
}

// QueryCustomerByName gets a customer with a given name.
func (c *Client) QueryCustomerByName(name string) (*Customer, error) {

	var r struct {
		QueryResponse struct {
			Customer []Customer
			TotalCount int
		}
	}
	err := c.query("SELECT * FROM Customer WHERE DisplayName = '"+
		strings.Replace(name, "'", "''", -1)+"'", &r)
	if err != nil {
		return nil, err
	}

	// var customers = make([]Customer, 0, r.QueryResponse.TotalCount)
	// for i := 0; i < r.QueryResponse.TotalCount; i += queryPageSize {
		// var page, err = c.fetchCustomerPage(i + 1)
		// if err != nil {
			// return nil, err
		// }
		// customers = append(customers, page...)
	// }
	return &r.QueryResponse.Customer[0], nil
}

// FetchCustomers gets the full list of Customers in the QuickBooks account.
func (c *Client) FetchCustomers() ([]Customer, error) {

	// See how many customers there are.
	var r struct {
		QueryResponse struct {
			TotalCount int
		}
	}
	err := c.query("SELECT COUNT(*) FROM Customer", &r)
	if err != nil {
		return nil, err
	}

	if r.QueryResponse.TotalCount == 0 {
		return make([]Customer, 0), nil
	}

	var customers = make([]Customer, 0, r.QueryResponse.TotalCount)
	for i := 0; i < r.QueryResponse.TotalCount; i += queryPageSize {
		var page, err = c.fetchCustomerPage(i + 1)
		if err != nil {
			return nil, err
		}
		customers = append(customers, page...)
	}
	return customers, nil
}

// Fetch one page of results, because we can't get them all in one query.
func (c *Client) fetchCustomerPage(startpos int) ([]Customer, error) {

	var r struct {
		QueryResponse struct {
			Customer      []Customer
			StartPosition int
			MaxResults    int
		}
	}
	q := "SELECT * FROM Customer ORDERBY Id STARTPOSITION " +
		strconv.Itoa(startpos) + " MAXRESULTS " + strconv.Itoa(queryPageSize)
	err := c.query(q, &r)
	if err != nil {
		return nil, err
	}

	// Make sure we don't return nil if there are no customers.
	if r.QueryResponse.Customer == nil {
		r.QueryResponse.Customer = make([]Customer, 0)
	}
	return r.QueryResponse.Customer, nil
}

// FetchCustomerByID returns a customer with a given ID.
func (c *Client) FetchCustomerByID(id string) (*Customer, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/customer/" + id
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	u.RawQuery = v.Encode()
	var req *http.Request
	req, err = http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	var res *http.Response
	res, err = c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("Got status code " + strconv.Itoa(res.StatusCode))
	}
	var r struct {
		Customer Customer
		Time     Date
	}
	err = json.NewDecoder(res.Body).Decode(&r)
	return &r.Customer, err
}

func (c *Client) getReport(url string) (*Report, error) {
	var req *http.Request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	var res *http.Response
	res, err = c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return c.parseReport(res)
}

func (c *Client) parseReport(res *http.Response) (*Report, error) {
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("Got status code " + strconv.Itoa(res.StatusCode))
	}
	var r Report
	err := json.NewDecoder(res.Body).Decode(&r)
	return &r, err
}

// SalesByCustomerID returns a the sales for the provided customer ID
func (c *Client) SalesByCustomerID(id string) (*Report, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/reports/CustomerSales/"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	// v.Add("start_date", "2020-01-01")
	// v.Add("end_date", "2022-01-01")
	v.Add("customer", id)
	u.RawQuery = v.Encode()

	return c.getReport(u.String())
}

func (c *Client) CashFlowByCustomerID(id string) (*Report, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/reports/CashFlow/"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	// v.Add("start_date", "2020-01-01")
	// v.Add("end_date", "2022-01-01")
	v.Add("customer", id)
	u.RawQuery = v.Encode()

	return c.getReport(u.String())
}

func (c *Client) ProfitAndLossByCustomerID(id string) (*Report, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/reports/ProfitAndLoss/"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	// v.Add("summarize_column_by", "Departments")
	// v.Add("start_date", "2020-01-01")
	// v.Add("end_date", "2022-01-01")
	v.Add("customer", id)
	u.RawQuery = v.Encode()

	return c.getReport(u.String())
}

func (c *Client) ProfitAndLossDetailsByCustomerID(id string) (*Report, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/reports/ProfitAndLossDetail/"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	// v.Add("summarize_column_by", "Departments")
	// v.Add("start_date", "2022-01-01")
	// v.Add("end_date", "2022-03-01")
	v.Add("customer", id)
	u.RawQuery = v.Encode()

	return c.getReport(u.String())
}

func (c *Client) TransactionList(customerID string) (*Report, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/reports/TransactionList/"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	// v.Add("summarize_column_by", "Departments")
	// v.Add("start_date", "2022-01-01")
	// v.Add("end_date", "2022-03-01")
	v.Add("group_by", "Transaction Type")
	v.Add("customer", customerID)
	u.RawQuery = v.Encode()

	return c.getReport(u.String())
}

func (c *Client) CustomerIncome(customerID string) (*Report, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/reports/CustomerIncome/"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	// v.Add("summarize_column_by", "Departments")
	// v.Add("start_date", "2022-01-01")
	// v.Add("end_date", "2022-03-01")
	v.Add("group_by", "Transaction Type")
	v.Add("customer", customerID)
	u.RawQuery = v.Encode()

	return c.getReport(u.String())
}

func (c *Client) AccountList(customerID string) (*Report, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/reports/AccountList/"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	// v.Add("summarize_column_by", "Departments")
	// v.Add("start_date", "2022-01-01")
	// v.Add("end_date", "2022-03-01")
	// v.Add("group_by", "Transaction Type")
	v.Add("customer", customerID)
	u.RawQuery = v.Encode()

	return c.getReport(u.String())
}

// CreateCustomer creates the given Customer on the QuickBooks server,
// returning the resulting Customer object.
func (c *Client) CreateCustomer(customer *Customer) (*Customer, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/customer"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	u.RawQuery = v.Encode()
	var j []byte
	j, err = json.Marshal(customer)
	if err != nil {
		return nil, err
	}
	var req *http.Request
	req, err = http.NewRequest("POST", u.String(), bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	var res *http.Response
	res, err = c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, parseFailure(res)
	}

	var r struct {
		Customer Customer
		Time     Date
	}
	err = json.NewDecoder(res.Body).Decode(&r)
	return &r.Customer, err
}

// UpdateCustomer updates the given Customer on the QuickBooks server,
// returning the resulting Customer object. It's a sparse update, as not all QB
// fields are present in our Customer object.
func (c *Client) UpdateCustomer(customer *Customer) (*Customer, error) {
	var u, err = url.Parse(string(c.Endpoint))
	if err != nil {
		return nil, err
	}
	u.Path = "/v3/company/" + c.RealmID + "/customer"
	var v = url.Values{}
	v.Add("minorversion", minorVersion)
	u.RawQuery = v.Encode()
	var d = struct {
		*Customer
		Sparse bool `json:"sparse"`
	}{
		Customer: customer,
		Sparse:   true,
	}
	var j []byte
	j, err = json.Marshal(d)
	if err != nil {
		return nil, err
	}
	var req *http.Request
	req, err = http.NewRequest("POST", u.String(), bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	var res *http.Response
	res, err = c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, parseFailure(res)
	}

	var r struct {
		Customer Customer
		Time     Date
	}
	err = json.NewDecoder(res.Body).Decode(&r)
	return &r.Customer, err
}
