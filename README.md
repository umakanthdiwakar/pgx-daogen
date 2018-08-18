# pgx-daogen
Generate Database access classes for postgres based on the pgx llibrary

GO with postgres is a fantastic platform to build micro-services. “pgx” (https://github.com/jackc/pgx), is the framework of choice for developing database access code for postgres and outperforms the pq library significantly.

However, given the procedural nature of pgx code, the issue facing a development team starting off with pgx, is the enourmous amount of boilerplate code needed to develop the database access objects. The purpose of pgx-daogen, is to generate well-defined data access objects, using pgx, from the meta-data in a specific postgres database. The aim here is to remove all dependecy on pgx-daogen from a runtime perspective and to only have well written pgx specific GO code. No hidden reflection based magic here.

### Usage :
```
pgx-daogen –init
```
This creates an empty **godao.config** file in the current directory. This is the file which pgx-daogen will use to generate all the code. A sample config file is given below. The fields are self-explanatory. **\*** can  be specified as the  value of the Tables array. Queries for which query-objects are to be generated can be specified in the **Queries** section. Recordsets and Query objects will then be generated for all tables/queries specified.
```
{
"Hostname" : "localhost",
"Dbname" : "mydatabase",
"Username" : "postgres",
"Password" : "admin",
"Tables" : [
	"table1"
],
"Queries" : [
	{
		"Name": "query1",
		"Query" : "select  * from table1"
	}
],
"PackageName" : "dao"
}
```

Once this configuration file is done, executing **pgx-daogen** will generate all the code under a directory, with the same name as the Packagename specified in the config file.

The generated recordset for a table named **Table1**, will consist of the following:
1.	**Table1VO** – A struct with GO native types, comprising the columns of the table.
2.	**Table1Rec** – A struct with pgx native types, comprising the columns of the table.
3.	**Table1Table** – The primary table object. This is the object that will be instantiated and used by the user (developer).
4.	**NewTable1** – Method to construct a new recordset. Prepares all the SQL statements. Takes one single parameter of type *DBConn*.
5.	**ScanRecord** – Method to read the column values from the Row object into the Rec struct.
6.	**SelectAll** – Method to fetch all rows of the underlying table
7.	**Select (key)** – Method to select a single row based on the primary key.
8.	**SelectFor (cond, parameters)** – Method to select one or more rows based on the condition.
9.	**Insert** – Method to insert into the table. Values are taken from the current VO object.
10.	**Update** – Method to update the column values. Values are taken from the current VO object.
11.	**FetchRecords** – Get all the rows that were previously selected either through Select, SelectAll or SelectFor into an array of VO objects. This is set in the VOs property as well as returned as a value object.
12.	**NextRow** – Method to read the next row from the rowset. This complements FetchRecords. While FetchRecords will get all the VOs as an array, NextRow will read the next row, convert into VO and set the current VO object. To be used in cases where the dataset is large and FetchRecords could swamp memory.
13.	**ConvertRecord2VO** – Method to convert from the native pgx types to GO types.
14.	**ConvertVO2Record** – Method to convert from GO types to native pgx types.
15.	*Getters and Setters* – One Getter and Setter for each column.

## Understanding how pgx-daogen saves a lot of developer effort.
The typical GO way of reading rows from tables involve:
1.	Executing a query
2.	Using Next to fetch the next row
3.	Issuing a Scan function to read in all the column values, passing a set of pointers.
4.	These addresses need to refer to pgx types, as passing GO types will throw an error if a null is present.
5.	Each pgx type has two important properties. A value property and a status property. The Status property needs to be checked for null before any mapping.
6.	Then these pgx types need to be converted into GO types.

**pgx-daogen** generated code, hides all these complexities from the developer. For example, when *NextRow* is executed, the generated code takes care of 
- fetching the next row, 
- scanning in all the column values, 
- checking for null, 
- mapping to native GO types and 
- setting these GO types to certain empty values if the underlying pgx type is null.
**pgx-daogen** also checks for column constraints and defaults and generates code accordingly. For example, if a date field has a default value of current date, then the generated *Insert* code does not bind these column values. To given another example, **pgx-daogen** identifies the primary key, and if this is auto-generated (if this is a serial type column), then it generates a RETURNING clause and sets the primary key VO property accordingly, post insert.

## Additional Features:
1. The generator supports **version columns**. When a table contains a column called **version**, the generated code will have the following changes:
   a.	In *Insert* method, the version column will be set to a value of 100.
   b.	Whenever the Update method is called, the version column value will be incremented by 1.
2. **Key generation**: To be able to generate alphanumeric keys automatically, the code generator supports a *seq_constants* table. This table needs to have three columns (list_table, sequence_name, constant_prefix). The framework also expects the sequences as given in seq_constants.sequence_name to be present in the database. Then the generated code will not accept user values for the primary key, but use the prefix and a 4 digit sequence to auto-generate the key.


## Using the generated recordset. 
Assuming that a recordset was generated for a table named **inbox**, with columns including "event_type" and "message_body". Then the generated code can be used as below. **CreateConnection** function comes from the **pgdb.go** file. Please do not forget to include pgdb.go in your project.  

```
  // CreateConnection comes from pgdb.go
    dbase, _ := CreateConnection("localhost", "WEBR", "postgres", "admin", 5)
    in := NewInbox (dbase)
    // SelectFor is used to execute the in-built select query for random conditions. However
    // only columns in the specific table will be fetched
    in.SelectFor("event_type = $1", "event1")
    // This is one way of looping through the rows. NextRow will scan into pgx types and then convert into the GO types in the VO
    for in.NextRow() {
        fmt.Println(in.Record)
        fmt.Println(in.VO)
    }
    // Using FetchRecords. This is an alternative method to using NextRow. Will fetch the entire rowset as an array of VOs.
    in.SelectAll()
    d := in.FetchRecords()
    // Both d and in.VOs will contain the array of VOs.
    fmt.Println(d)
    fmt.Println(in.VOs)

    // Illustrating Update
    in.SetMessageBody("hello this is changed")
    in.Update()
    dbase.Close()
```

