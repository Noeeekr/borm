### Migrations
1. `Done` Migrate Environment: Recreate Flag
2. `Done` Migrate Environment: Ignore flag
3. `Done` Migrate Environment: Drop flag
4. `Done` Migrate Environment: Connect to database environment when migrating an environment
5. Migrate Relations: Default tag
6. Migrate Relations: Improve enum reflection

### Operations
1. `Unnecessary: The user can easily create/manage its own enums` Store enum type values and do type check in fields using it so it can only use enum values
2. `Done: Users can use their own enums without type convertion` Change RegisterEnum to accept any and check if it is a string type
3. Create database error array to implement better error managment through error subscription
4. `Done: Users can order by ascending or descending` Add order method to query builder.
4. [LAST ADDED] Add limit method to query builder.
5. [LAST ADDED] And Clause for where.

### Package
1. `Done` Make types public to use. Will need a huge refactor in folder structure and codebase logic.

### Documentation
1. Comments specifying which errors each function sends
