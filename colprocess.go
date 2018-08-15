package main

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/pgtype"
)

type DBGen struct {
	dbconn           *DBase
	initializeCalled bool
}

func (p *DBGen) Initialize(conn *DBase) {
	p.dbconn = conn
	p.initializeCalled = true
}

type GoColInfo struct {
	goColName    string
	voType       string
	recType      string
	nullValue    string
	pgValueField string
	pgTypeCast   string
}

type ColDesc struct {
	TableSchema   string
	TableName     string
	ColumnName    string
	DataType      string
	ColumnDefault string
	Constraints   string
	IsNullable    bool
	goInfo        GoColInfo
}

type ColSummary struct {
	insertCols    []int
	primaryCols   []int
	selectCols    []int
	returningCols []int
	updateCols    []int
}

type TableMap struct {
	colDesc        []ColDesc
	colSummary     ColSummary
	Sequencename   string
	Sequenceprefix string
	HasTime        bool
	HasVersion     bool
}

var typeMap = map[string]GoColInfo{
	"character": GoColInfo{
		voType:       "string",
		recType:      "pgtype.Varchar",
		nullValue:    `""`,
		pgValueField: "String",
		pgTypeCast:   "string(",
	},
	"text": GoColInfo{
		voType:       "string",
		recType:      "pgtype.Text",
		nullValue:    `""`,
		pgValueField: "String",
		pgTypeCast:   "string(",
	},
	"time": GoColInfo{
		voType:       "string",
		recType:      "pgtype.Timestamptz",
		nullValue:    `""`,
		pgValueField: "Time",
		pgTypeCast:   "time.Parse(time.RFC3339,",
	},
	"date": GoColInfo{
		voType:       "string",
		recType:      "pgtype.Timestamptz",
		nullValue:    `""`,
		pgValueField: "Time",
		pgTypeCast:   "time.Parse(time.RFC3339,",
	},
	"boolean": GoColInfo{
		voType:       "bool",
		recType:      "pgtype.Bool",
		nullValue:    "false",
		pgValueField: "Bool",
		pgTypeCast:   "bool(",
	},
	"integer": GoColInfo{
		voType:       "int",
		recType:      "pgtype.Int4",
		nullValue:    "-1",
		pgValueField: "Int",
		pgTypeCast:   "int32(",
	},
	"bigint": GoColInfo{
		voType:       "int64",
		recType:      "pgtype.Int8",
		nullValue:    "-1",
		pgValueField: "Int",
		pgTypeCast:   "int64(",
	},
	"bytea": GoColInfo{
		voType:       "[]byte",
		recType:      "pgtype.Bytea",
		nullValue:    "[]byte{}",
		pgValueField: "Bytes",
		pgTypeCast:   "[]byte(",
	},
}

func getGoColInfo(colType string) GoColInfo {
	for k, v := range typeMap {
		if strings.HasPrefix(colType, k) {
			return v
		}
	}
	v := typeMap["character"]
	return v
}

func CreateNewTablemap() *TableMap {
	m := TableMap{}
	m.colDesc = make([]ColDesc, 0)
	m.colSummary = ColSummary{}
	m.colSummary.insertCols = make([]int, 0)
	m.colSummary.primaryCols = make([]int, 0)
	m.colSummary.returningCols = make([]int, 0)
	m.colSummary.selectCols = make([]int, 0)
	m.colSummary.updateCols = make([]int, 0)
	return &m
}
func ProcessColMetadata(db *DBase) map[string]*TableMap {
	//db, err := CreateConnection("localhost", dbname, username, password, numconns)
	conn := db.ConnPool
	tableMap := make(map[string]*TableMap, 0)
	rows, err := conn.Query(`
		select c.table_schema,c.table_name,c.column_name,c.data_type, 
		c.column_default,c.is_nullable::bool, 
			(select array_to_string(array_agg(tc.constraint_type::text),',') from 
			information_schema.key_column_usage kc join information_schema.table_constraints tc on 
			kc.constraint_name = tc.constraint_name 
			where kc.table_schema=c.table_schema and kc.table_name=c.table_name 
			and kc.column_name=c.column_name) 
		 as constraints 
		from information_schema.columns c where c.table_schema = 'public' 
		order by c.table_name,c.ordinal_position
		`)
	if err != nil {
		fmt.Println("***ERROR*** : Reading information schema. Error = ", err)
		return nil
	}
	//fmt.Println("Error returned:", err)
	defer rows.Close()

	columnNum := 0
	for rows.Next() {
		trec := ColDesc{}

		var columnDefaultVC, constraintsVC pgtype.Varchar

		err := rows.Scan(&trec.TableSchema, &trec.TableName, &trec.ColumnName,
			&trec.DataType, &columnDefaultVC, &trec.IsNullable, &constraintsVC)
		if err != nil {
			fmt.Println("***ERROR***", "Generate", err)
			return nil
		}
		trec.goInfo = getGoColInfo(trec.DataType)
		trec.goInfo.goColName = convertCase(trec.ColumnName)
		trec.ColumnDefault = ""
		if columnDefaultVC.Status == pgtype.Present {
			trec.ColumnDefault = columnDefaultVC.String
		}
		trec.Constraints = ""
		if constraintsVC.Status == pgtype.Present {
			trec.Constraints = constraintsVC.String
		}

		/*fmt.Println(trec.TableName, ",", trec.ColumnName, ",", trec.DataType, ",",
		trec.ColumnDefault, ",", trec.Constraints, ",", trec.IsNullable)*/
		if _, ok := tableMap[trec.TableName]; !ok {
			tableMap[trec.TableName] = CreateNewTablemap()
			err = conn.QueryRow("select sequence_name, constant_prefix from seq_constants where list_table = $1", trec.TableName).Scan(&tableMap[trec.TableName].Sequencename, &tableMap[trec.TableName].Sequenceprefix)
			if err != nil {
				tableMap[trec.TableName].Sequencename = ""
				tableMap[trec.TableName].Sequenceprefix = ""
			}
			columnNum = 0
		}
		mp := tableMap[trec.TableName]
		if trec.goInfo.pgValueField == "Time" {
			mp.HasTime = true
		}
		if trec.ColumnName == "version" {
			mp.HasVersion = true
		}
		mp.colDesc = append(mp.colDesc, trec)
		mp.colSummary.selectCols = append(mp.colSummary.selectCols, columnNum)
		if strings.Contains(trec.Constraints, "PRIMARY") {
			mp.colSummary.primaryCols = append(mp.colSummary.primaryCols, columnNum)
		} else {
			mp.colSummary.updateCols = append(mp.colSummary.updateCols, columnNum)
		}
		if len(trec.ColumnDefault) == 0 {
			mp.colSummary.insertCols = append(mp.colSummary.insertCols, columnNum)
		}
		if len(trec.ColumnDefault) > 0 || strings.Contains(trec.Constraints, "PRIMARY") {
			mp.colSummary.returningCols = append(mp.colSummary.returningCols, columnNum)
		}
		columnNum++
	}
	return tableMap
}
