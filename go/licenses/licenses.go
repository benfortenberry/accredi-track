package licenses

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/benfortenberry/accredi-track/utils"
	"github.com/gin-gonic/gin"
)

type License struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"createdBy"`
	InUseBy   string `json:"inUseBy"`
}

func Get(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var licenses []License

	query := `SELECT id, name,
 ( SELECT IFNULL(JSON_ARRAYAGG(employeeId) ,"")
FROM employeeLicenses el where el.licenseId = l.id and el.deleted is null ) as inUseBy
FROM licenses l where deleted IS NULL and createdBy =?`
	rows, err := db.Query(query, userSubStr)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query licenses"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var lic License
		if err := rows.Scan(
			&lic.ID, &lic.Name, &lic.InUseBy,
		); err != nil {
			fmt.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan license data"})
			return
		}
		licenses = append(licenses, lic)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating over license rows"})
		return
	}

	c.IndentedJSON(http.StatusOK, licenses)
}

func Post(db *sql.DB, c *gin.Context) {

	userSubStr, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	var lic License
	if err := c.BindJSON(&lic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prepare the SQL statement for inserting
	query := `
        INSERT INTO licenses(
            name, createdBy
        ) VALUES (?, ?)
    `

	// Execute the query
	result, err := db.Exec(query,
		lic.Name, userSubStr,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert license"})
		return
	}

	// Get the ID of the newly inserted
	id, err := result.LastInsertId()
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve inserted license ID"})
		return
	}

	// Respond with the ID of the newly created
	c.JSON(http.StatusOK, gin.H{"message": "License inserted successfully", "id": id})
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
        UPDATE licenses
        SET deleted = current_timestamp()
        WHERE id = ?
    `

	// Execute the query
	result, err := db.Exec(query, id)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete license"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// Respond with a success message
	c.JSON(http.StatusOK, gin.H{"message": "License deleted successfully"})
}

func Put(db *sql.DB, c *gin.Context) {

	_, ok := utils.GetUserSub(c)
	if !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "User Not Found"})
		return
	}

	// Get the ID from the URL parameter
	id := c.Param("id")

	var lic License
	if err := c.BindJSON(&lic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prepare the SQL statement for updating
	query := `
        UPDATE licenses
        SET name = ?
        WHERE id = ?
    `

	// Execute the query
	_, err := db.Exec(query,
		lic.Name, id,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update license"})
		return
	}

	// Check if any rows were affected
	// rowsAffected, err := result.RowsAffected()
	// if err != nil {
	// 	fmt.Println("Error: ", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve affected rows"})
	// 	return
	// }

	// if rowsAffected == 0 {
	// 	c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
	// 	return
	// }

	var updatedLicense License
	getQuery := `
		 SELECT id, name
		 FROM licenses
		 WHERE id = ?
	 `
	err = db.QueryRow(getQuery, id).Scan(
		&updatedLicense.ID,
		&updatedLicense.Name,
	)
	if err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated license"})
		return
	}

	// Respond with the updated data
	c.JSON(http.StatusOK, updatedLicense)

}
