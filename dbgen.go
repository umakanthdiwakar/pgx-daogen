package main

import (
	"fmt"
)

func generateProgram(tableName string, tableMap *TableMap) {
	tableName1 := convertCase(tableName)
	cols := tableMap.colDesc
	colSumm := tableMap.colSummary
	sequenceName := tableMap.Sequencename
	sequencePrefix := tableMap.Sequenceprefix

	//=========   Generate the imports ===========
	{
		timeImport := ""
		if tableMap.HasTime {
			timeImport = `"time"`
		}

		ff(`import (
	"errors"
	"fmt"
	%s

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)
`, timeImport)
	}
	//-------------------------------------

	//=========   Create the VO type  ===================
	{
		ff("// %sVO - Value object format to be used in code\n", tableName1)
		ff("type %sVO struct {\n", tableName1)

		for _, v := range cols {
			ff("\t%-30s%s\n", v.goInfo.goColName, v.goInfo.voType)
		}
		ff("}\n\n")
	}
	//---------------------------------------------

	//==========    Generate the Record type     =================
	{
		ff("// %sRec - Record format using native types for database interaction\n", tableName1)
		ff("type %sRec struct {\n", tableName1)
		for _, v := range cols {
			ff("\t%-30s%s\n", v.goInfo.goColName, v.goInfo.recType)
		}
		ff("}\n\n")
	}
	//--------------------------------------------------

	//==============   Generate the main table type   ===============
	{
		ff("// %sTable - the primary table object\n", tableName1)
		ff("type %sTable struct {\n", tableName1)
		ff("\t%-30s%s\n", "DBconn", "*DBase")
		ff("\t%-30s%sRec\n", "Record", tableName1)
		ff("\t%-30s%sVO\n", "VO", tableName1)
		ff("\t%-30s[]%sVO\n", "VOs", tableName1)
		ff("\t%-30s%s\n", "CurrentRows", "*pgx.Rows")
		ff("\t%-30s%s\n", "CurrentRow", "*pgx.Row")
		ff("\t%-30s%s\n", "singleRowSelected", "bool")
		ff("\t%-30s%s\n", "Statements", "map[string]string")
		ff("}\n\n")
	}
	//---------------------------------------------------

	//==========     Generate Initialize function ==============
	{
		ff("//Initialize - function to initialize the base struct\n")
		ff(`func (t *%sTable) Initialize(dbconn *DBase) {
	t.DBconn = dbconn
	t.Statements = map[string]string{
`, tableName1)
		s1 := ""
		s2 := ""
		s1, s2 = generateSelectStatement(tableName, tableMap)
		ff("\t\t\"%sSelectAll\": \"%s\",\n", tableName1, s1)
		ff("\t\t\"%sSelect\": \"%s%s\",\n", tableName1, s1, s2)
		s1 = generateInsertStatement(tableName, tableMap)
		ff("\t\t\"%sInsert\": \"%s\",\n", tableName1, s1)
		s1 = generateUpdateStatement(tableName, tableMap)
		ff("\t\t\"%sUpdate\": \"%s\",\n", tableName1, s1)
		ff("\t}\n")
		ff("\tt.Record = %sRec {}\n", tableName1)
		ff("\tt.VO = %sVO {}\n", tableName1)
		ff("\tt.singleRowSelected = false\n")
		ff("}\n\n")
	}
	//-----------------------------------------------------

	//==========     Generate Reinitialize function ==============
	{
		ff("//Reinitialize - function to reinitialize the VO and Rec\n")
		ff(`func (t *%sTable) Reinitialize() {
`, tableName1)
		ff("\tt.Record = %sRec {}\n", tableName1)
		ff("\tt.VO = %sVO {}\n", tableName1)
		ff("\tt.singleRowSelected = false\n")
		ff("}\n\n")
	}
	//-----------------------------------------------------

	//=========      Generate ScanRecord function    =============
	{
		ff("// ScanRecord - Scans the current Row into the Record variable\n")
		scanRecordList := ""
		for i, v := range cols {
			if i > 0 {
				scanRecordList += ", "
			}
			temp := fmt.Sprintf("&t.Record.%s", v.goInfo.goColName)
			scanRecordList += temp
		}
		ff("func (t *%sTable) ScanRecord() (*%sRec, error) {\n", tableName1, tableName1)
		ff("\tvar err error\n")
		ff("\tif t.singleRowSelected {\n")
		ff("\t\terr = t.CurrentRow.Scan(%s)\n", scanRecordList)

		ff("\t} else {\n\t\terr = t.CurrentRows.Scan(%s)\n\t}\n\treturn &t.Record, err\n}\n\n", scanRecordList)

	}
	//-------------------------------------------------------

	//=============   two functions - PrepareStatement4Key and ExecuteQuery ===========
	{
		ff(`// PrepareStatement4Key - Given key, fetches and prepares statement
func (t *%sTable) PrepareStatement4Key(queryKey string) error {
	c := t.DBconn.ConnPool
	if _, ok := t.Statements[queryKey]; !ok {
		return errors.New("***ERROR*** - ExecuteQuery - Given query key (" + queryKey + ") not found!")
	}
	_, err := c.Prepare(queryKey, t.Statements[queryKey])
	if err != nil {
		fmt.Println("***ERROR***", "Preparing Select All", err)
		return err
	}
	return nil
}

// ExecuteQuery - common function to execute given keyed query and return results
func (t *%sTable) ExecuteQuery(queryKey string, args ...interface{}) error {
	if err := t.PrepareStatement4Key(queryKey); err != nil {
		return err
	}
	c := t.DBconn.ConnPool
	if t.CurrentRows != nil {
		t.CurrentRows.Close()
	}
	rows, err := c.Query(queryKey, args...)
	if err != nil {
		fmt.Println("***ERROR***", "Query Select All", err)
		return err
	}
	t.CurrentRows = rows
	return nil
}

`, tableName1, tableName1)
	}
	//---------------------------------------------------------

	//======== Two more - SelectAll and Select
	{
		primaryKeyParamList := ""
		primaryKeyParamList1 := ""
		{
			for i, subs := range colSumm.primaryCols {
				col := cols[subs]
				if i > 0 {
					primaryKeyParamList += ", "
					primaryKeyParamList1 += ", "
				}
				temp := fmt.Sprintf("key%d %s", i+1, col.goInfo.voType)
				primaryKeyParamList += temp
				temp = fmt.Sprintf("key%d", i+1)
				primaryKeyParamList1 += temp
			}

		}
		ff(`// SelectAll - issues a select on the table. Sets the CurrentRows element
// to the returned rows. These will now be available via NextRow
func (t *%sTable) SelectAll() error {
	return t.ExecuteQuery("%sSelectAll")
}

// Select - issues a select on the table for key. Sets the CurrentRows element
// to the returned rows. These will now be available via NextRow
func (t *%sTable) Select(%s) error {
	return t.ExecuteQuery("%sSelect", %s)
}

`, tableName1, tableName1, tableName1, primaryKeyParamList, tableName1, primaryKeyParamList1)

	}
	//-------------------------------------------------------------

	//============   Generate SelectFor    =======================
	{
		ff(`// SelectFor - first param is the WHERE clause without WHERE.
// Optional arguments can be passed. Sets the CurrentRows element
// to the returned rows. These will now be available via NextRow
func (t *%sTable) SelectFor(whereCond string, args ...interface{}) error {
	c := t.DBconn.ConnPool
	query := t.Statements["%sSelectAll"]
	if len(whereCond) > 0 {
		query += " WHERE "
		query += whereCond
	}
	if t.CurrentRows != nil {
		t.CurrentRows.Close()
	}
	rows, err := c.Query(query, args...)
	if err != nil {
		fmt.Println("***ERROR***", "Query SelectFor", err)
		return err
	}
	t.CurrentRows = rows
	return nil
}

`, tableName1, tableName1)
	}
	//------------------------------------------------------------

	//===== conditionally generate the Genkey function    =============
	genKeyCallingCode := ""
	{
		primaryKeyCol := ""
		primaryKeySubs := 0
		if len(colSumm.primaryCols) > 0 {
			primaryKeySubs = colSumm.primaryCols[0]
		}
		primaryKeyCol = cols[primaryKeySubs].goInfo.goColName

		if len(sequenceName) > 0 {
			ff(`// Genkey - used to generate primary key if entry present in seq_constants
func (t *%sTable) Genkey() error {
	c := t.DBconn.ConnPool
	nextVal := 0
	err := c.QueryRow("select nextval('%s')").Scan(&nextVal)
	if err != nil {
		fmt.Println("*** Scan ***", err)
		return err
	}
	`, tableName1, sequenceName)
			ff("key := fmt.Sprintf(\"%%s%%04d\", \"%s\", nextVal)\n", sequencePrefix)
			ff(`	t.VO.%s = key
	return nil
}

`, primaryKeyCol)
			genKeyCallingCode = "t.Genkey()"
		}
	}
	//---------------------------------------------------------------

	//=========== Generate Insert function =========================
	{
		assignToVersion := ""
		if tableMap.HasVersion {
			assignToVersion = "t.VO.Version = 100"
		}
		ff(`// Insert - Used to insert record. The Record struct needs to
// be filled before calling Insert
func (t *%sTable) Insert() error {
	queryKey := "%sInsert"
	var err error
	if err = t.PrepareStatement4Key(queryKey); err != nil {
		fmt.Println(err)
		return err
	}
	c := t.DBconn.ConnPool
	%s
	%s
	t.ConvertVO2Record()
	r := &t.Record
`, tableName1, tableName1, genKeyCallingCode, assignToVersion)
		if len(colSumm.returningCols) > 0 {
			ff("\trow := c.QueryRow(queryKey")
		} else {
			ff("\t_, err = c.Exec(queryKey")
		}
		for _, v := range colSumm.insertCols {
			col := cols[v]
			temp := fmt.Sprintf(", &r.%s", col.goInfo.goColName)
			ff(temp)
		}
		ff(")\n")
		if len(colSumm.returningCols) > 0 {
			ff("\terr = row.Scan(")
			for i, v := range colSumm.returningCols {
				col := cols[v]
				if i > 0 {
					ff(", ")
				}
				ff("&r.%s", col.goInfo.goColName)
			}
			ff(")\n")

		}
		ff("\treturn err\n}\n\n")
	}
	//---------------------------------------------

	//=======   Generate Update function   =======================
	{
		// Generate the update function
		ff(`// Update - Used to update record. The Record struct needs to
// be filled before calling Update
func (t *%sTable) Update() error {
	queryKey := "%sUpdate"
	if err := t.PrepareStatement4Key(queryKey); err != nil {
		fmt.Println(err)
		return err
	}
	c := t.DBconn.ConnPool
	t.ConvertVO2Record()
	r := t.Record
`, tableName1, tableName1)
		ff("\t_, err := c.Exec(queryKey")
		for _, v := range colSumm.updateCols {
			if cols[v].ColumnName != "version" {
				ff(", ")
				ff("&r.%s", convertCase(cols[v].ColumnName))
			}
		}
		for _, v := range colSumm.primaryCols {
			ff(", ")
			ff("&r.%s", convertCase(cols[v].ColumnName))
		}
		ff(")\n\treturn err\n}\n\n")
	}
	//----------------------------------------------------------------

	//==============     Generate FetchRows   =====================
	{
		ff(`// FetchRecords - Fetches all records into VOs object
// based on the current CurrentRows
func (t *%sTable) FetchRecords() []%sVO {
	t.VOs = []%sVO{}
	for t.NextRow() {
		v := t.VO
		t.VOs = append(t.VOs, v)
	}
	return t.VOs
}

`, tableName1, tableName1, tableName1)
	}
	//------------------------------------------------------------------

	//================  Generate NextRow function    ===========
	{
		// Generate NextRow function
		ff(`// NextRow - Used to scroll through the contained Rows
func (t *%sTable) NextRow() bool {
	if t.CurrentRows != nil {
		ret := t.CurrentRows.Next()
		if ret {
			t.ScanRecord()
			t.ConvertRecord2VO()
		}
		return ret
	}
	return false
}

`, tableName1)
	}
	//--------------------------------------------------------------

	//=============     Generate ConvertRecord2VO function  ===============
	{
		ff(`// ConvertRecord2VO - Convert pgtype types to VO go types
func (t *%sTable) ConvertRecord2VO() *%sVO {
`, tableName1, tableName1)
		for _, v := range cols {
			toString := ""
			if v.goInfo.pgValueField == "Time" {
				toString = ".String()"
			}
			ff("\tif t.Record.%s.Status == pgtype.Present {\n", v.goInfo.goColName)
			ff("\t\tt.VO.%s = %s(t.Record.%s.%s%s)\n",
				v.goInfo.goColName, v.goInfo.voType,
				v.goInfo.goColName, v.goInfo.pgValueField, toString)
			ff("\t} else {\n")
			ff("\t\tt.VO.%s = %s(%s)\n", v.goInfo.goColName, v.goInfo.voType,
				v.goInfo.nullValue)
			ff("\t}\n\n")
		}
		ff("\treturn &t.VO\n")
		ff("}\n\n")
	}
	//-------------------------------------------------------------------

	//==========    Generate ConvertVO2Record   ==================
	{
		ff(`// ConvertVO2Record - Convert GO types to pgtype types
func (t *%sTable) ConvertVO2Record() *%sRec {
`, tableName1, tableName1)
		for _, v := range cols {
			errOption := ""
			if v.goInfo.pgValueField == "Time" {
				errOption = ", _ "
			}
			ff("\tt.Record.%s.Status = pgtype.Present\n", v.goInfo.goColName)
			ff("\tt.Record.%s.%s%s = %st.VO.%s)\n\n",
				v.goInfo.goColName, v.goInfo.pgValueField, errOption,
				v.goInfo.pgTypeCast, v.goInfo.goColName)
		}
		ff("\treturn &t.Record\n")
		ff("}\n\n")
	}
	//------------------------------------------------------------------------

	//======   Generate getters and setters  ======================
	{
		for _, v := range cols {
			// Generate the getter
			toString := ""
			errOption := ""
			if v.goInfo.pgValueField == "Time" {
				toString = ".String()"
				errOption = ", _ "
			}
			ff("func (t *%sTable) Get%s () %s {\n", tableName1, v.goInfo.goColName, v.goInfo.voType)
			ff("\tif t.Record.%s.Status == pgtype.Present {\n", v.goInfo.goColName)
			ff("\t\tt.VO.%s = %s(t.Record.%s.%s%s)\n", v.goInfo.goColName, v.goInfo.voType,
				v.goInfo.goColName, v.goInfo.pgValueField, toString)
			ff("\t} else {\n")
			ff("\t\tt.VO.%s = %s(%s)\n", v.goInfo.goColName,
				v.goInfo.voType, v.goInfo.nullValue)
			ff("\t}\n\n")
			ff("\treturn t.VO.%s\n", v.goInfo.goColName)
			ff("}\n\n")

			// Now generate the setter
			ff("func (t *%sTable) Set%s (value %s) {\n", tableName1,
				v.goInfo.goColName, v.goInfo.voType)
			ff("\tt.VO.%s = value\n", v.goInfo.goColName)
			ff("\tt.Record.%s.Status = pgtype.Present\n", v.goInfo.goColName)
			ff("\tt.Record.%s.%s%s = %svalue)\n", v.goInfo.goColName,
				v.goInfo.pgValueField, errOption, v.goInfo.pgTypeCast)
			ff("}\n\n")
		}
	}
	//---------------------------------------------------------

}

func generateSelectStatement(tableName string,
	tableMap *TableMap) (string, string) {
	colDesc := tableMap.colDesc
	colSumm := tableMap.colSummary
	selectStatement := "SELECT "
	for i, subs := range colSumm.selectCols {
		if i > 0 {
			selectStatement += ", "
		}
		c := colDesc[subs]
		selectStatement += c.ColumnName
	}
	selectStatement += " FROM "
	selectStatement += tableName
	whereCond := ""
	if len(colSumm.primaryCols) > 0 {
		whereCond = " WHERE "
		pos := 1
		for _, subs := range colSumm.primaryCols {
			col := colDesc[subs]
			if pos > 1 {
				whereCond += " AND "
			}
			temp := fmt.Sprintf(" %s = $%d ", col.ColumnName, pos)
			whereCond += temp
			pos++
		}
	}
	return selectStatement, whereCond
}

func generateInsertStatement(tableName string, tableMap *TableMap) string {
	colDesc := tableMap.colDesc
	colSumm := tableMap.colSummary
	statement := fmt.Sprintf("INSERT INTO %s ( ", tableName)
	for i, subs := range colSumm.insertCols {
		if i > 0 {
			statement += ", "
		}
		col := colDesc[subs]
		statement += col.ColumnName
	}
	statement += ") VALUES ("
	for i := 1; i <= len(colSumm.insertCols); i++ {
		if i > 1 {
			statement += ", "
		}
		temp := fmt.Sprintf("$%d", i)
		statement += temp
	}
	statement += ")"
	if len(colSumm.returningCols) > 0 {
		statement += " RETURNING "
		for i, subs := range colSumm.returningCols {
			col := colDesc[subs]
			if i > 0 {
				statement += ", "
			}
			temp := fmt.Sprintf(" %s ", col.ColumnName)
			statement += temp
		}
	}

	return statement
}

func generateUpdateStatement(tableName string, tableMap *TableMap) string {
	colDesc := tableMap.colDesc
	colSumm := tableMap.colSummary
	statement := fmt.Sprintf("UPDATE %s SET ", tableName)
	versionEncountered := 0
	for i, subs := range colSumm.updateCols {
		if i > 0 {
			statement += ", "
		}
		col := colDesc[subs]
		temp := fmt.Sprintf("%s = $%d", col.ColumnName, i+1-versionEncountered)
		if col.ColumnName == "version" {
			versionEncountered = 1
			temp = fmt.Sprintf("%s = %s + 1", col.ColumnName, col.ColumnName)
		}
		statement += temp
	}
	statement += " WHERE "
	for i, subs := range colSumm.primaryCols {
		if i > 0 {
			statement += " AND "
		}
		col := colDesc[subs]
		temp := fmt.Sprintf("%s = $%d", col.ColumnName, i+len(colSumm.updateCols)+1)
		statement += temp
	}

	return statement
}
