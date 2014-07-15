StateDB
=======

StateDB is an application-level checkpointing library that differentiates between mutable and immutable attributes in the application state. 
The application programmer defines the mutable and immutable attributes of each state-entry (type that needs to be saved), and the library uses that information to checkpoint immutable data only *once*, and checkpoint *only* the mutable data in every subsequent checkpoint.

### Target Applications
The library does not track the changes in the mutable state-entries, rather it performs a naive checkpoint of *every* mutable attribute. This makes the library well suited for applications that perform frequent updates to mutable attributes (eg. simulations where positions and directions are constantly updated). However, this approach is not optimal in applications where a large number of the mutable attributes are in-frequently updated, such as databases etc. 

### Use
The library requires the application programmer to register each state-entry manually, and provides iterators for restoration purposes. 
