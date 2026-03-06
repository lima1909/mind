<div align="center">

# Mind 

[![Build Status](https://img.shields.io/github/actions/workflow/status/lima1909/mind/ci.yaml?style=for-the-badge)](https://github.com/lima1909/mind/actions)
![License](https://img.shields.io/github/license/lima1909/mind?style=for-the-badge)
[![Stars](https://img.shields.io/github/stars/lima1909/mind?style=for-the-badge)](https://github.com/lima1909/mind/stargazers)

</div>

`Mind (Multi Index List)` means quickly finding list items using indexes to improve query/filter operations for lists.

<div>
⚠️ <strong>Mind is in a very early stage of development and can change!</strong>
</div>
 
### Advantage

The fast access can be achieved by using different methods, like;

- hash tables
- indexing
- ...

### Disadvantage

- it is more memory required. In addition to the user data, data for the _hash_, _index_ are also stored.
- the write operation are slower, because for every wirte operation is an another one (for storing the index data) necessary

