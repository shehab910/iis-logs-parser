Dear Rana,

Thank you for the detailed and clear description. I have implemented all the recommended changes and new features for Tasks 3 and 4. You can find the latest updates pushed to the [same GitHub repository](https://github.com/shehab910/iis-logs-parser).

I look forward to your valuable feedback.

Below, you’ll find the instructions to run the project along with some important notes.

## How to run

1. Make google app password

   > Note: When I tried to use my personal email it wouldn't work

   - Go to [Google Account](https://myaccount.google.com/u/1/apppasswords)
   - Make a new app and copy the password

1. Set the password in the environment variable

   - In the project go to `./.env.local`
   - Replace `FROM_EMAIL` and `FROM_EMAIL_PASSWORD` with your email and password

1. Make sure Postgres is running and has the same configuration as in `./.env.local`

1. Run the project

   ```bash
   go run main.go
   ```

1. (optional) use the attached postman collection to test the endpoints

# Notes & Thoughts

### Authentication flow change

I have made some minor modifications to the authentication process flow, please find attached the modified flow diagram.

### DB Denormalization

I was thinking about denormalizing the db by adding the user id directly to log files, this is justified (imo) because almost all request (db calls) will need to verify the ownership of the log file.

Regards,
