package employeeLicesnses

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/benfortenberry/accredi-track/utils"
	"github.com/gin-gonic/gin"
)

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

type EmployeeLicenseInsert struct {
	ID         int    `json:"id"`
	EmployeeID int    `json:"employeeId"`
	LicenseID  int    `json:"licenseId"`
	IssueDate  string `json:"issueDate"`
	ExpDate    string `json:"expDate"`
}

func Get(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	id := c.Param("id")

	var employeeLicenses []EmployeeLicense
	query := `
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
	where el.employeeId = ?
	 and el.deleted IS NULL
	and el.createdBy = ?

	`
	rows, err := db.Query(query, id, userSubStr)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query employee licenses"})
		return
	}
	defer rows.Close()

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

	if err := rows.Err(); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating over employee license rows"})
		return
	}

	c.IndentedJSON(http.StatusOK, employeeLicenses)
}

func Post(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var lic EmployeeLicenseInsert
	if err := c.BindJSON(&lic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prepare the SQL statement for inserting
	query := `
        INSERT INTO employeeLicenses(
            employeeId,
			licenseId,
			issueDate,
			expDate,
			createdBy
        ) VALUES (?, ?, ?, ?, ?)
    `

	// Execute the query
	result, err := db.Exec(query,
		lic.EmployeeID,
		lic.LicenseID,
		lic.IssueDate,
		lic.ExpDate,
		userSubStr,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert employee license"})
		return
	}

	// Get the ID of the newly inserted
	id, err := result.LastInsertId()
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve inserted employee license ID"})
		return
	}

	// Respond with the ID of the newly created
	c.JSON(http.StatusOK, gin.H{"message": "Employee License inserted successfully", "id": id})
}

func Delete(db *sql.DB, c *gin.Context) {

	_, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	// Get the ID from the URL parameter
	id := c.Param("id")

	// Prepare the SQL statement for deleting
	query := `
        UPDATE employeeLicenses
        SET deleted = current_timestamp()
        WHERE id = ?
    `

	// Execute the query
	result, err := db.Exec(query, id)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee license"})
		return
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve affected rows"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee license not found"})
		return
	}

	// Respond with a success message
	c.JSON(http.StatusOK, gin.H{"message": "Employee License deleted successfully"})
}

func Put(db *sql.DB, c *gin.Context) {

	_, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	// Get the ID from the URL parameter
	id := c.Param("id")

	var lic EmployeeLicenseInsert
	if err := c.BindJSON(&lic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prepare the SQL statement for updating
	query := `
        UPDATE employeeLicenses
		SET 
			licenseId = ?,
			issueDate = ?,
			expDate = ?
        WHERE id = ?
    `

	// fmt.Println(lic.LicenseID, lic.IssueDate, lic.ExpDate, id)

	// Execute the query
	_, err := db.Exec(query,
		lic.LicenseID,
		lic.IssueDate,
		lic.ExpDate,
		id,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee license"})
		return
	}

	var updatedLicense EmployeeLicense
	getQuery := `
		 SELECT el.id,
	el.employeeId ,
	el.licenseId,
	el.issueDate,
	el.expDate,
	e.firstName,
	e.lastName,
	e.phone1,
	e.email,
	l.name as licenseName
		from
	employeeLicenses el
left join employees e on
	el.employeeId = e.id
left join licenses l on
	el.licenseId = l.id
		 WHERE el.id = ? and el.deleted IS NULL
	 `
	err = db.QueryRow(getQuery, id).Scan(
		&updatedLicense.ID,
		&updatedLicense.EmployeeID,
		&updatedLicense.LicenseID,
		&updatedLicense.IssueDate,
		&updatedLicense.ExpDate,
		&updatedLicense.FirstName,
		&updatedLicense.LastName,
		&updatedLicense.Phone1,
		&updatedLicense.Email,
		&updatedLicense.LicenseName,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated employee license"})
		return
	}

	// Respond with the updated data
	c.JSON(http.StatusOK, updatedLicense)

}
