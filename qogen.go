package main

import (
	"fmt"
)

func genQueryObject(qInfo QueryInfo, cols []ColDesc) {
	goQueryName := convertCase(qInfo.Name)

	//=========   Generate the imports ===========
	{

		ff("%s", `import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)
`)
	}
	//-------------------------------------

	//=========   Create the VO type  ===================
	{
		ff("// %sVO - Value object format to be used in code\n", goQueryName)
		ff("type %sVO struct {\n", goQueryName)

		for _, v := range cols {
			ff("\t%-30s%s\n", v.goInfo.goColName, v.goInfo.voType)
		}
		ff("}\n\n")
	}
	//---------------------------------------------

	//==========    Generate the Record type     =================
	{
		ff("// %sRec - Record format using native types for database interaction\n", goQueryName)
		ff("type %sRec struct {\n", goQueryName)
		for _, v := range cols {
			ff("\t%-30s%s\n", v.goInfo.goColName, v.goInfo.recType)
		}
		ff("}\n\n")
	}
	//--------------------------------------------------

	//==============   Generate the main query object   ===============
	{
		ff("// %s - the primary query object\n", goQueryName)
		ff("type %s struct {\n", goQueryName)
		ff("\t%-30s%s\n", "DBconn", "*DBase")
		ff("\t%-30s%sRec\n", "Record", goQueryName)
		ff("\t%-30s%sVO\n", "VO", goQueryName)
		ff("\t%-30s[]%sVO\n", "VOs", goQueryName)
		ff("\t%-30s%s\n", "CurrentRows", "*pgx.Rows")
		ff("\t%-30s%s\n", "Query", "string")
		ff("\t%-30s%s\n", "QueryName", "string")
		ff("\t%-30s%s\n", "isInitializeCalled", "bool")
		ff("}\n\n")
	}
	//---------------------------------------------------

	//==========     Generate Initialize function ==============
	{
		ff("//Initialize - function to initialize the base struct\n")
		ff(`func (t *%s) Initialize(dbconn *DBase) {
t.DBconn = dbconn
`, goQueryName)
		ff("\tt.Record = %sRec {}\n", goQueryName)
		ff("\tt.VO = %sVO {}\n", goQueryName)
		ff("\tt.VOs = [] %sVO {}\n", goQueryName)
		ff("\tt.QueryName = \"%s\"\n", qInfo.Name)
		ff("\tt.Query = \"%s\"\n", qInfo.Query)
		ff("}\n\n")
	}
	//-----------------------------------------------------

	//===== Generate ExecuteQuery   ===================
	{
		ff(`func (t *%s) ExecuteQuery(dbconn *DBase, args ...interface{}) error {
	t.Initialize(dbconn)
	c := t.DBconn.ConnPool
	_, err := c.Prepare(t.QueryName, t.Query)
	if err != nil {
		fmt.Println("***ERROR***", "Preparing Query", t.Query, err)
		return err
	}
	rows, err := c.Query(t.QueryName, args...)
	if err != nil {
		fmt.Println("***ERROR***", "Executing Query", t.Query, err)
		return err
	}
	t.CurrentRows = rows
	return nil
}

`, goQueryName)
	}
	//-----------------------------------------------------

	//==========      Generate FetchRecords    ====================
	{
		ff(`func (t *%s) FetchRecords(dbconn *DBase, args ...interface{}) ([]%sVO, error) {
	err := t.ExecuteQuery(dbconn, args...)
	if err != nil {
		return nil, err
	}
	t.VOs = []%sVO{}
	for t.NextRow() {
		t.VOs = append(t.VOs, t.VO)
		t.VO = %sVO{}
	}
	return t.VOs, nil
}

`, goQueryName, goQueryName, goQueryName, goQueryName)
	}
	//-----------------------------------------------------------

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
		ff("func (t *%s) ScanRecord() (*%sRec, error) {\n", goQueryName, goQueryName)
		ff("\tvar err error\n")
		ff("\terr = t.CurrentRows.Scan(%s)\n\treturn &t.Record, err\n}\n\n", scanRecordList)

	}
	//-------------------------------------------------------

	//================  Generate NextRow function    ===========
	{
		// Generate NextRow function
		ff(`// NextRow - Used to scroll through the contained Rows
func (t *%s) NextRow() bool {
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

`, goQueryName)
	}
	//--------------------------------------------------------------

	//=============     Generate ConvertRecord2VO function  ===============
	{
		ff(`// ConvertRecord2VO - Convert pgtype types to VO go types
func (t *%s) ConvertRecord2VO() *%sVO {
`, goQueryName, goQueryName)
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

}

func getQueryObject(dbconn *DBase, queryInfo QueryInfo) []ColDesc {
	query := queryInfo.Query + " LIMIT 0"
	rows, err := dbconn.ConnPool.Query(query)

	if err != nil {
		fmt.Println("((", query, "))")
		fmt.Println("***ERROR***", "executing query. Query=", queryInfo.Name, "Error=", err)
		return nil
	}
	defer rows.Close()
	fdesc := rows.FieldDescriptions()
	cols := []ColDesc{}

	for _, v := range fdesc {
		//fmt.Println(v.DataType, v.DataTypeName, v.DataTypeSize, v.Name, v.Table)
		col := ColDesc{}
		col.ColumnName = v.Name
		col.DataType = v.DataTypeName
		col.goInfo = getGoColInfo(col.DataType)
		col.goInfo.goColName = convertCase(col.ColumnName)
		cols = append(cols, col)
	}

	return cols
}
