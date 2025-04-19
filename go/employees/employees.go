package employees

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/benfortenberry/accredi-track/utils"
	"github.com/gin-gonic/gin"
)

type Employee struct {
	ID           int    `json:"id"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Phone1       string `json:"phone1"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	LicenseCount int    `json:"licenseCount"`
}

func Get(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var employees []Employee
	query := (`
	SELECT 
    e.id AS employeeId,
    e.firstName,
    e.lastName,
    e.phone1,
    e.email,
    CASE 
        WHEN EXISTS (
            SELECT 1 
            FROM employeeLicenses el 
            WHERE el.employeeId = e.id 
              AND el.expDate < CURDATE() 
              AND el.deleted IS NULL
        ) THEN 'Expired'
        ELSE 'Active'
    END AS status,
	( SELECT COUNT(*) AS cnt
FROM employeeLicenses el where el.employeeId = e.id and el.deleted is null ) as licenseCount
FROM 
    employees e
where e.deleted is null and createdBy = ? `)
	rows, err := db.Query(query, userSubStr)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query employees"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var emp Employee
		if err := rows.Scan(
			&emp.ID, &emp.FirstName, &emp.LastName,

			&emp.Phone1, &emp.Email, &emp.Status, &emp.LicenseCount,
		); err != nil {
			fmt.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan employee data"})
			return
		}

		//encodedID := encoding.EncodeID(emp.ID)
		// emp.ID = 0                 // Clear the original ID
		//emp.EmployeeID = encodedID // Add the encoded ID to the response

		employees = append(employees, emp)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating over rows"})
		return
	}

	c.IndentedJSON(http.StatusOK, employees)
}

func GetSingle(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	// Get the employee ID from the URL parameter
	id := c.Param("id")

	// Prepare the SQL query to retrieve the employee
	query := `
        SELECT id, firstName, lastName, phone1, email
        FROM employees
        WHERE id = ? AND deleted IS NULL and createdBy = ?
    `

	// Create an Employee struct to hold the result
	var emp Employee

	// Execute the query
	err := db.QueryRow(query, id, userSubStr).Scan(
		&emp.ID,
		&emp.FirstName,
		&emp.LastName,
		&emp.Phone1,
		&emp.Email,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no rows are found, return a 404 error
			c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		} else {
			// For other errors, return a 500 error
			fmt.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve employee"})
		}
		return
	}

	// Respond with the employee data
	c.JSON(http.StatusOK, emp)
}

func Post(db *sql.DB, c *gin.Context) {
	// Bind the JSON payload to an Employee struct

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var emp Employee
	if err := c.BindJSON(&emp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prepare the SQL statement for inserting an employee
	query := `
        INSERT INTO employees (
            firstName, lastName, 
            phone1, email, createdBy
        ) VALUES (?, ?, ?, ?, ?)
    `

	// Execute the query
	result, err := db.Exec(query,
		emp.FirstName, emp.LastName, emp.Phone1, emp.Email, userSubStr,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert employee"})
		return
	}

	// Get the ID of the newly inserted employee
	id, err := result.LastInsertId()
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve inserted employee ID"})
		return
	}

	// Respond with the ID of the newly created employee
	c.JSON(http.StatusOK, gin.H{"message": "Employee inserted successfully", "id": id})
}

func Delete(db *sql.DB, c *gin.Context) {

	_, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	// Get the employee ID from the URL parameter
	id := c.Param("id")

	// Prepare the SQL statement for deleting an employee
	query := `
        UPDATE employees
        SET deleted = current_timestamp()
        WHERE id = ?;
    `

	// Execute the query
	_, err := db.Exec(query, id)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee"})
		return
	}

	queryEmployeeLicenses := `
        UPDATE employeeLicenses
        SET deleted = current_timestamp()
        WHERE employeeId = ?;
    `

	// Execute the query
	_, err2 := db.Exec(queryEmployeeLicenses, id)
	if err2 != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee licenses"})
		return
	}

	// Respond with a success message
	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted successfully"})
}

func Put(db *sql.DB, c *gin.Context) {

	_, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	// Get the employee ID from the URL parameter
	id := c.Param("id")

	// Bind the JSON payload to an Employee struct
	var emp Employee
	if err := c.BindJSON(&emp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prepare the SQL statement for updating an employee
	query := `
        UPDATE employees
        SET firstName = ?, lastName = ?, phone1 = ?, email = ?
        WHERE id = ?
    `

	// Execute the query
	_, err := db.Exec(query,
		emp.FirstName, emp.LastName, emp.Phone1, emp.Email, id,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee"})
		return
	}

	// Query the updated employee data
	var updatedEmployee Employee
	getQuery := `
		 SELECT id, firstName, lastName, phone1, email
		 FROM employees
		 WHERE id = ?
	 `
	err = db.QueryRow(getQuery, id).Scan(
		&updatedEmployee.ID,
		&updatedEmployee.FirstName,
		&updatedEmployee.LastName,
		&updatedEmployee.Phone1,
		&updatedEmployee.Email,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated employee"})
		return
	}

	// Respond with the updated employee data
	c.JSON(http.StatusOK, updatedEmployee)

}
