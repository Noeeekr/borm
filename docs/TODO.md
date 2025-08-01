### Migrations
1. Migrate Environment: Recreate Flag
2. Migrate Environment: Ignore flag
3. Migrate Environment: Drop flag
4. Migrate Environment: Connect to database environment when migrating an environment
5. Migrate Relations: Default tag
6. Migrate Relations: Improve enum reflection

### Operations
1. Store enum type values and do type check in fields using it so it can only use enum values
2. [LAST ADDED] Change RegisterEnum to accept any and check if it is a string type
3. [LAST ADDED] Create database error array to implement better error managment through error subscription
 
### Package
1. Make types public to use. Will need a huge refactor in folder structure and codebase logic.

### Documentation
1. Comments specifying which errors each function sends
