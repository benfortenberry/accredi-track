package employees

import (
	"database/sql"
	"fmt"
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
	ComplianceRate        float32 `json:"complianceRate"`
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

func isPastDate(date time.Time) bool {
	// Get current date, truncated to remove time
	now := time.Now().Truncate(24 * time.Hour)
	// Truncate input date to remove time
	date = date.Truncate(24 * time.Hour)

	return date.Before(now)
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

	// 	queryTotalExpiredEmployeeLicenses := (`
	// select count(*) from employeeLicenses el where deleted
	// is null and expDate < CURRENT_DATE() and createdBy = ?
	// `)

	// 	queryTotalExpiringSoonEmployeeLicenses := (`
	// select count(*) from employeeLicenses el where deleted
	// is null and expDate < CURDATE() + INTERVAL 1 DAY and expDate > CURDATE()  and createdBy = ?
	// `)

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
`)

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

	// total employee Licenses

	var employeeLicenses []EmployeeLicense

	rows, err2 := db.Query(queryEmployeeLicenses, userSubStr)

	// err2 := db.QueryRow(queryTotalEmployeeLicenses, userSubStr).Scan(
	// 	&metrics.TotalEmployeeLicenses,
	// )

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

	layout := "2006-01-02"

	for i := 0; i < len(employeeLicenses); i++ {
		// Access each element using the index
		element := expiredEmployeeLicenses[i]
		parsedTime, err := time.Parse(layout, element.ExpDate)

		if err != nil {
			fmt.Println("Error parsing time:", err)
			return
		}

		if isPastDate(parsedTime) {
			expiredEmployeeLicenses = append(expiredEmployeeLicenses, element)
		}
		// Perform operations with the element
		//fmt.Println("Index:", i, "Value:", element)
	}

	metrics.ExpiredCount = len(expiredEmployeeLicenses)
	fmt.Println((expiredEmployeeLicenses))

	// // total expiring soon
	// err3 := db.QueryRow(queryTotalExpiringSoonEmployeeLicenses, userSubStr).Scan(
	// 	&metrics.ExpiringSoon,
	// )

	// if err3 != nil {
	// 	if err3 == sql.ErrNoRows {
	// 		// If no rows are found, return a 404 error
	// 		c.JSON(http.StatusNotFound, gin.H{"error": "Data not found"})
	// 	} else {
	// 		// For other errors, return a 500 error
	// 		fmt.Println("Error: ", err3)
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard metrics"})
	// 	}
	// 	return
	// }

	// // Respond with the employee data
	// c.JSON(http.StatusOK, metrics)
}
