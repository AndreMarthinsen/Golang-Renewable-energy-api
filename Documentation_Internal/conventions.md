# Project Conventions
Draft 1.0: 31. march 2023
## Code Documentation
### Constants:
- No 'magic values' are allowed, meaning no un-explained hard coded literals.
- Make use of constants for all URLs, internal and external.
- Constants relating to a single package should be placed in a file of their own in the package directory.
- Multi package constants should be in a package of their own

### Commenting:
- Structs should have at least a minimal comment explaining its intended use case and.
- Never leave the function comment describing its functionality for later. Assume your code should be ready to be handed off to another member at any moment.
- Code should be self documenting through naming of functions and variables, comments only providing context for non-trivial logic.
- Maximize information, minimize verbosity. Leave out what can be inferred from context and naming.
- If you are unsure if a segment of code is trivial, err on the side of caution and add a brief explanation.
### Naming:
- Use camelCase for non-exporting entities, PascalCase for exporting ones.
- Avoid single letter names unless it is a variable with a short lifespan with an obvious purpose. If in doubt, use descriptive names.
- Avoid 'noise' in names, such as customers_slice where slice can be inferred from object type.
- Avoid abbreviations or acronyms.

## Git-usage

### Issues
- Issues should be linked to a Milestone.
- Issues should have a description with enough information that other team members will be able to understand what the issue is about without having to ask follow-up questions.

### Branching
- Create a branch per issue you are working on.
- Never work directly on the main branch.

### Commits
- Address one problem per commit. 
- Commit messages should be linked to issues through use of keywords such as 'Closes #x', 'Relates to #x", etc.

### Merging
- The other team members should review and provide feedback on your work prior to any merge into main being done.
- Ensure your code compiles before you merge one branch into another unless it is for the purpose of joining two issues for further development.

### Cleanliness
- Do not push any redundant files to the code-base, such as compiled binaries or IDE related files.

