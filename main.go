package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

var configFileName = "godao.config"

func genConfigFile() {
	f, err := os.Create(configFileName)
	if err != nil {
		fmt.Println("Error creating config file ...", err)
		os.Exit(1)
	}
	writer := bufio.NewWriter(f)
	fmt.Fprintf(writer, `
{
"Hostname" : "localhost",
"Dbname" : "mydb",
"Username" : "dbuser",
"Password" : "dbpwd",
"Tables" : [
	"*"
],
"Queries" : [
	{
		"Name": "query1",
		"Query" : "select  col1, col2 from table1 where col1 = some_condition"
	}
],
"PackageName" : "main"
}
`)
	writer.Flush()
	f.Close()
}

func processGodaoFile() {
	v := GetGenData(configFileName)
	os.MkdirAll(v.PackageName, 0755)
	dbase, _ := CreateConnection(v.Hostname, v.Dbname, v.Username, v.Password, 5)
	defer dbase.Close()
	fmt.Printf("Connection worked!\n")
	t := ProcessColMetadata(dbase)
	if t == nil {
		os.Exit(1)
	}
	fmt.Println("***   GENERATING RECORD SETS   ***")
	for tableName, tableMap := range t {
		tableTobeProcessed := false
		if v.Tables[0] == "*" {
			tableTobeProcessed = true
		} else {
			for _, table := range v.Tables {
				if tableName == table {
					tableTobeProcessed = true
					break
				}
			}
		}
		if !tableTobeProcessed {
			continue
		}
		fmt.Print(tableName)
		globalfp, _ := os.Create(v.PackageName + "/" + tableName + "Recordset.go")
		_global_writer = bufio.NewWriter(globalfp)
		ff("package %s\n\n", v.PackageName)
		generateProgram(tableName, tableMap)
		globalfp.Close()
		fmt.Println(" ... Completed.")
	}

	fmt.Println("\n***   GENERATING QUERY OBJECTS   ***")
	for _, q := range v.Queries {
		cols := getQueryObject(dbase, q)
		if cols == nil {
			continue
		}
		goQueryName := convertCase(q.Name)
		fmt.Print(goQueryName)
		globalfp, _ := os.Create(v.PackageName + "/" + goQueryName + "QO.go")
		_global_writer = bufio.NewWriter(globalfp)
		ff("package %s\n\n", v.PackageName)
		genQueryObject(q, cols)
		globalfp.Close()
		fmt.Println(" ... Completed.")
	}
}

func main() {
	initPtr := flag.Bool("init", false, "Create empty config file")
	flag.Parse()
	if *initPtr {
		genConfigFile()
		os.Exit(0)
	}

	if _, err := os.Stat(configFileName); !os.IsNotExist(err) {
		processGodaoFile()
	} else {
		fmt.Println("Error opening config file. Use --init option to create empty file")
		os.Exit(1)
	}
}
