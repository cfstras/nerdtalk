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
Returns an _AuthState_.

### /logout/ ###
Invalidates the auth token for the current user, if any.  
Clears auth cookies.  
Returns an _AuthState_

### AuthState ###

    {
      "AuthState": 5,
      "StringState": "Valid"
    }
    
Possible Values:

 - 0, _Unknown_
 - 1, _InvalidUser_
 - 2, _InvalidID_
 - 4, _WrongPassword_
 - 5, _WrongToken_
 - 6, _Valid_

**TODO**: These expose too much info. Make Auth failures less verbose.

## Parameters ##

### ?redirect=true ###
If this is specified, a successful _add_ request will not output its results in JSON but redirect to the appropriate page containing the new content.  
This is used for non-javascript browsers.

## /get/ ##

### /get/post/<post id> ###
fetches a post from the database, adressed by its uid.

### /get/posts/<thread id> ###
fetches posts from the database belonging to a given thread uid.

### /get/user/<user id> ###
fetches info about a user from the database, adressed by his/her uid.

### /get/thread/<thread id> ###
fetches info about a thread from the database, adressed by its uid.

### /get/posts/<thread id> ###
fetches posts from the database which belong to a thread adressed by the given uid.

### /get/threads/ ###
fetches all threads from the database.

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

### /add/like/<post ID> ##
Adds a _Like_ to the given post. Returns the new post _Like_ list.
**TODO** what happens if you double-like?
