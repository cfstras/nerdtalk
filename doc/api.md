# REST API #

Accessible via `/get/` and `/add/`. Prints JSON.

## Authentication ##
Happens via these cookies:

    nerdtalk-uid: "<user id>"
    nerdtalk-token: "<auth token>"

They can be aquired with `/login/`.

### /login/ ###
Accepts two POST values, `User` and `Pass`, each containing strings.  
On success, auth cookies are set.  
Returns an AuthState.

### AuthState ###

    {
      "AuthState": 5,
      "StringState": "Valid"
    }
    
Possible Values:

 - 0, Unknown
 - 1, InvalidUser
 - 2, InvalidID
 - 4, WrongPassword
 - 5, WrongToken
 - 6, Valid

**TODO**: These expose too much info. Make Auth failures less verbose.

## /get/ ##

### /get/post/<id> ###
fetches a post from the database, adressed by its uid.

### /get/user/<id> ###
fetches info about a user from the database, adressed by his/her uid.

### /get/thread/<id> ###
fetches info about a thread from the database, adressed by its uid.

### /get/posts/<id> ###
fetches posts from the database which belong to a thread adressed by the given uid.

### /get/threads/ ###
fetches all threads from the database.

### /get/threads/<id> ###
fetches threads from the database which belong to a user adressed by the given uid.

## /add/ ##

### /add/post/<id> ###
Inserts a post into the thread with the given id.
The post text has to be delivered in a POST-attribute named `Text`

### /add/thread/ ###
Adds a new thread.
The title of the thread is expected as POST-attribute `Title`, the text for the first post in `Text`.

### /add/user/ ###
Adds a new user.
**TODO**
