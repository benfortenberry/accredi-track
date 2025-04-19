package employees

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/benfortenberry/accredi-track/utils"
	"github.com/gin-gonic/gin"
)

type Metrics struct {
	TotalEmployees        int     `json:"totalEmployees"`
	ExpiredCount          int     `json:"expiredCount"`
	ExpiringSoon          int     `json:"expiringSoon"`
	LicenseAvg            float32 `json:"licenseAvg"`
	NotificationCount     int     `json:"notificationCount"`
	ComplianceRate        float64 `json:"complianceRate"`
	TotalEmployeeLicenses int     `json:"totalEmployeeLicenses"`
}

type EmployeeLicense struct {
	ID          int    `json:"id"`
	EmployeeID  int    `json:"employeeId"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Phone1      string `json:"phone1"`
	Email       string `json:"email"`
	LicenseName string `json:"licenseName"`
	LicenseID   int    `json:"licenseId"`
	IssueDate   string `json:"issueDate"`
	ExpDate     string `json:"expDate"`
}

type LicenseChartData struct {
	Count       int    `json:"count"`
	LicenseName string `json:"licenseName"`
}

type LicenseExpiringChartData struct {
	Count int    `json:"count"`
	Month string `json:"licenseName"`
}

func isPastDate(date time.Time) bool {
	// Get current date, truncated to remove time
	now := time.Now().Truncate(24 * time.Hour)
	// Truncate input date to remove time
	date = date.Truncate(24 * time.Hour)

	return date.Before(now)
}

func isAlmostPastDate(date time.Time) bool {
	// Get current date, truncated to remove time
	soon := time.Now().Truncate(24*time.Hour).AddDate(0, 0, 30)
	fmt.Println(soon)

	// Truncate input date to remove time
	date = date.Truncate(24 * time.Hour)

	return date.Before(soon)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func Get(db *sql.DB, c *gin.Context) {

	// Convert userSub to a string
	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var metrics Metrics

	queryTotalEmployees := (`
	select count(*) as count from employees e 
where e.deleted is null and createdBy = ? `)

	//total employees
	err1 := db.QueryRow(queryTotalEmployees, userSubStr).Scan(
		&metrics.TotalEmployees,
	)

	if err1 != nil {
		if err1 == sql.ErrNoRows {
			// If no rows are found, return a 404 error
			c.JSON(http.StatusNotFound, gin.H{"error": "Data not found"})
		} else {
			// For other errors, return a 500 error
			fmt.Println("Error: ", err1)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard metrics"})
		}
		return
	}

	// get all employee Licenses

	queryEmployeeLicenses := (`
	select
		el.id,
		el.employeeId ,
		el.licenseId,
		el.issueDate,
		el.expDate,
		l.name  as licenseName
	from
		employeeLicenses el
	left join employees e on
		el.employeeId = e.id
	left join licenses l on
		el.licenseId = l.id
		 and el.deleted IS NULL
		and el.createdBy = ?
	where el.deleted is null
	`)

	var employeeLicenses []EmployeeLicense

	rows, err2 := db.Query(queryEmployeeLicenses, userSubStr)

	if err2 != nil {
		if err2 == sql.ErrNoRows {
			// If no rows are found, return a 404 error
			c.JSON(http.StatusNotFound, gin.H{"error": "Data not found"})
		} else {
			// For other errors, return a 500 error
			fmt.Println("Error: ", err2)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard metrics"})
		}
		return
	}

	for rows.Next() {
		var lic EmployeeLicense
		if err := rows.Scan(
			&lic.ID, &lic.EmployeeID, &lic.LicenseID,
			&lic.IssueDate, &lic.ExpDate,
			&lic.LicenseName,
		); err != nil {
			fmt.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan employee license data"})
			return
		}

		employeeLicenses = append(employeeLicenses, lic)
	}

	defer rows.Close()

	var expiredEmployeeLicenses []EmployeeLicense
	var expiringSoonEmployeeLicenses []EmployeeLicense
	layout := "2006-01-02"

	for i := 0; i < len(employeeLicenses); i++ {
		element := employeeLicenses[i]

		parsedTime, err := time.Parse(layout, element.ExpDate)

		if err != nil {
			fmt.Println("Error parsing time:", err)
			return
		}

		if isPastDate(parsedTime) {
			expiredEmployeeLicenses = append(expiredEmployeeLicenses, element)
		} else if isAlmostPastDate(parsedTime) {
			expiringSoonEmployeeLicenses = append(expiringSoonEmployeeLicenses, element)

		}

	}

	totalActive := len(employeeLicenses) - len(expiredEmployeeLicenses)
	metrics.ComplianceRate = toFixed(float64(totalActive)/float64(len(employeeLicenses)), 2) * 100
	metrics.ExpiredCount = len(expiredEmployeeLicenses)
	metrics.ExpiringSoon = len(expiringSoonEmployeeLicenses)
	metrics.TotalEmployeeLicenses = len(employeeLicenses)
	metrics.LicenseAvg = float32(len(employeeLicenses)) / float32(metrics.TotalEmployees)

	//notifications last 30 days

	queryNotifications := (`
	select count(*) as count from notifications
where userSub = ? `)

	err3 := db.QueryRow(queryNotifications, userSubStr).Scan(
		&metrics.NotificationCount,
	)

	if err3 != nil {
		if err3 == sql.ErrNoRows {
			// If no rows are found, return a 404 error
			c.JSON(http.StatusNotFound, gin.H{"error": "Data not found"})
		} else {
			// For other errors, return a 500 error
			fmt.Println("Error: ", err3)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard metrics"})
		}
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func GetLicenseChartData(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var licenseChartData []LicenseChartData
	query := (`
	SELECT COUNT(el.id) as count, l.name 
FROM employeeLicenses el 
left join licenses l on el.licenseId = l.id 
where el.deleted is null and el.expDate > CURDATE() and el.createdby = ?
GROUP BY l.name ;`)
	rows, err := db.Query(query, userSubStr)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query license chart"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var licChartData LicenseChartData
		if err := rows.Scan(
			&licChartData.Count, &licChartData.LicenseName,
		); err != nil {
			fmt.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan license chart data"})
			return
		}

		licenseChartData = append(licenseChartData, licChartData)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating over rows"})
		return
	}

	c.IndentedJSON(http.StatusOK, licenseChartData)
}

func GetExpiredLicenseChartData(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var licenseChartData []LicenseChartData
	query := (`
	SELECT COUNT(el.id) as count, l.name 
FROM employeeLicenses el 
left join licenses l on el.licenseId = l.id 
where el.deleted is null and el.expDate < CURDATE() and el.createdby = ?
GROUP BY l.name ;`)
	rows, err := db.Query(query, userSubStr)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query expired license chart"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var licChartData LicenseChartData
		if err := rows.Scan(
			&licChartData.Count, &licChartData.LicenseName,
		); err != nil {
			fmt.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan expired license chart data"})
			return
		}

		licenseChartData = append(licenseChartData, licChartData)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating over rows"})
		return
	}

	c.IndentedJSON(http.StatusOK, licenseChartData)
}

func GetExpiringsByMonth(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var licenseChartData []LicenseExpiringChartData
	query := (`
	
    
   SELECT
	DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 1 MONTH), '%M') AS month,
	COUNT(*) AS count
FROM
	employeeLicenses
WHERE
	expDate BETWEEN CURDATE() AND DATE_ADD(CURDATE(), INTERVAL 1 MONTH)
	and deleted is null  and createdBy= ?
union all 
       SELECT
	DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 2 MONTH), '%M') AS month,
	COUNT(*) AS count
FROM
	employeeLicenses
WHERE
	expDate BETWEEN DATE_ADD(CURDATE(), INTERVAL 1 MONTH) AND DATE_ADD(CURDATE(), INTERVAL 2 MONTH)
	and deleted is null  and createdBy= ?
union all
       SELECT
	DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 3 MONTH), '%M') AS month,
	COUNT(*) AS count
FROM
	employeeLicenses
WHERE
	expDate BETWEEN DATE_ADD(CURDATE(), INTERVAL 2 MONTH) AND DATE_ADD(CURDATE(), INTERVAL 3 MONTH)
	and deleted is null  and createdBy= ?
	union all
  SELECT
	DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 4 MONTH), '%M') AS month,
	COUNT(*) AS count
FROM
	employeeLicenses
WHERE
	expDate BETWEEN DATE_ADD(CURDATE(), INTERVAL 3 MONTH) AND DATE_ADD(CURDATE(), INTERVAL 4 MONTH)
	and deleted is null and createdBy= ?
		union all
  SELECT
	DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 5 MONTH), '%M') AS month,
	COUNT(*) AS count
FROM
	employeeLicenses
WHERE
	expDate BETWEEN DATE_ADD(CURDATE(), INTERVAL 4 MONTH) AND DATE_ADD(CURDATE(), INTERVAL 5 MONTH)
	and deleted is null  and createdBy= ?
`)
	rows, err := db.Query(query, userSubStr)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query expiring license chart"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var licChartData LicenseExpiringChartData
		if err := rows.Scan(
			&licChartData.Count, &licChartData.Month,
		); err != nil {
			fmt.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan expiring license chart data"})
			return
		}

		licenseChartData = append(licenseChartData, licChartData)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating over rows"})
		return
	}

	c.IndentedJSON(http.StatusOK, licenseChartData)
}
