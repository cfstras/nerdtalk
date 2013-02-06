REST API
========

Accessible via /get/ and /add/. Prints JSON.

/get/
=====

/get/post/<id>
--------------
fetches a post from the database, adressed by its uid.

/get/user/<id>
--------------
fetches info about a user from the database, adressed by his/her uid.

/get/thread/<id>
----------------
fetches info about a thread from the database, adressed by its uid.

/get/posts/<id>
---------------
fetches posts from the database which belong to a thread adressed by the given uid.

/get/threads/
-------------
fetches all threads from the database.

/get/threads/<id>
-----------------
fetches threads from the database which belong to a user adressed by the given uid.

/add/
=====

//TODO