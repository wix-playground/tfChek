tfChek
======

This is a web application for Terrafrom continuous integration

_*tfChek*_ performs check of the applied terraform resources.
 * If check was successful it makes a pull request and tries to automatically merge it. 
 If it is impossible to merge the changes automatically, reviewer should make it manually. 
 GitHub will send an appropriate email notification.
 * In case of failed check the GitHub issue will be created and assigned to the author of the commit



Changelog

*0.0.2*
>This version and all prior versions does not have any working version deployed
 It supports basic operations only. Nothing is well tested.

*0.0.3*
> Can copy task output

*0.0.4*
> Add buttons bar

*0.1.0*
> Application is able to create GitHub pull requests and GitHub issues. Send notifications via GitHub. Also it can automatically merge the pull requests.
