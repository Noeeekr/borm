Borm (Better Oriented Relational Mapper) is meant to be a safe and faster approach to simple database tasks. Managing boilerplate code for you as much as possible, without losing performance.

# TOPICS
___
## MIGRATIONS:
    // Register user and database for migration. Migrating a database also migrate its user.
    myDatabaseUser := borm.RegisterUser("my_database_user", "my_database_user_passwd")
    myDatabase := borm.RegisterDatabase("my_databas_name", myDatabaseUser)

    // Register tables and enums for migration.
    enum := myDatabase.RegisterEnum("fruits", "banana", "apple", "pineapple")
    table := myDatabase.RegisterTable(MyTableStruct{}).NeedRoles(enum)
    
    // Needs to connect to postgres on first time
    borm.Connect("postgres", "postgres-Host", "postgres-Password", "postgres-User")

    borm.Settings().Migrations().Enable()
    borm.Settings().Migrations().RecreateExisting()

    // To migrate databases, users, etc.
    borm.MigrateEnvironment()
    
    // To migrate tables, types, etc.
    borm.MigrateRelations()
___
## TAGS
```
      Usage:
        // Table name can be changed in the RegisterTable function
        type MyDatabaseTable struct {
          // Becomes a field of table MyDatabaseTable with name myPrivateField
          myNonIgnoredPrivateField string 
          
          // Ignored
          myIgnoredPrivateField string ``borm:"(IGNORE)"

          // Becomes a field of table MyDatabaseTable with name my_public_field
          MyPublicField string `borm:"(NAME, my_public_field) (CONSTRAINTS, NOT NULL)"`
        }
```
```
  (CONSTRAINTS, string) Defines the constraints of the field.
      Ex: (CONSTRAINTS, PRIMARY KEY)
    
  (NAME, string) Defines the name of the field. If not present name will be the field name to lowercase.
      Ex: (NAME, my_field_name)

  (TYPE, string) Defines the type of the field in case it is not implemented on reflection.
      Ex: (TYPE, VARCHAR(555))

  (FOREIGN KEY, primary_key_table_name, primary_key_field_name) Defines the field as a foreign key
      Ex: (FOREIGN KEY, users, id)

  (IGNORE) Ignores a field completely for all borm operations.
```
___
## Operations
```
  myServiceDatabase := borm.RegisterDatabase("my_service_database_name", myServiceDatabaseOwnerUser)
  TableProducts := myServiceDatabase.RegisterTable(Products{}).Name("products")

  --

  q := TableProducts.Select("product_name", "product_quantity")
  q.Where(q.Field("product_quantity").Equals(10))
  q.Scanner(scannerFunc)
```
