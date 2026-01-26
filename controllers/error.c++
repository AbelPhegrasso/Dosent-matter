public static void SendErrorNotification(string mainCaseNumber, StringBuilder accountNamesList)
        {
            Log.Information($"[{mainCaseNumber}] [EXCEPT] SendErrorNotification Function in Catch Error.");
            Log.Information($"[{mainCaseNumber}] [EXCEPT] Data accountNamesList in list {accountNamesList}");
            // Split accountNamesList by line to get each account name
            string accountNamesString = accountNamesList.ToString();
            string[] accountNames = accountNamesString.Split(new[] { Environment.NewLine }, StringSplitOptions.RemoveEmptyEntries);
            Log.Information($"[{mainCaseNumber}] [EXCEPT] Data accountNames to send {accountNames}");

            StringBuilder bodyBuilder = new StringBuilder();
            bodyBuilder.AppendLine("<p>เรียน ผู้ใช้งาน,</p><p>บริษัทฯ ขอแจ้งว่า <strong>ไม่สามารถส่งรายงานการโอนเงิน</strong> ของระบบ Online Payment Services (OPS)</p><p>โดยรายงานดังกล่าวเป็นรายงานการชำระเงินของบริษัทดังต่อไปนี้:</p><ol>");

            // Loop through each account name and append it in the email body with numbering
            for (int i = 0; i < accountNames.Length; i++)
            {
                bodyBuilder.AppendLine($"<li>{accountNames[i]}</li>");
            }

            bodyBuilder.AppendLine("</ol><p>ขอขอบคุณมา ณ ที่นี้</p><p>INET Online Payment Service</p>");

            string body = bodyBuilder.ToString();

            MailMessage message = new MailMessage();

            try
            {
                string subject = $"[Case No.[{mainCaseNumber}][EXCEPT] ส่งรายงานการรับเงินประจำวันไม่สำเร็จ";
                message.To.Add(new MailAddress(ConfigurationManager.AppSettings["error_to_mail"]));
                message.From = new MailAddress("no-reply@thaidotcompayment.co.th", "INET Online Payment Service");
                message.Subject = subject;
                message.Body = body;
                message.BodyEncoding = Encoding.UTF8;
                message.IsBodyHtml = true;
            }
            catch (FormatException)
            {
                Log.Error($"[{mainCaseNumber}] [EXCEPT] Invalid error_to_e-mail address is null or wrong format");
            }


            SmtpClient smtp = new SmtpClient(ConfigurationManager.AppSettings["smtp_Client"]);
            smtp.Port = int.Parse(ConfigurationManager.AppSettings["smtp_Port"]);
            smtp.UseDefaultCredentials = false;
            smtp.EnableSsl = false;
            smtp.Credentials = new NetworkCredential(ConfigurationManager.AppSettings["smtp_Email_Credential"], ConfigurationManager.AppSettings["smtp_Password_Credential"]);

            smtp.Send(message);
            Log.Information($"[{mainCaseNumber}] [EXCEPT] Message Data: From: {message.From}, To: {string.Join(", ", message.To.Select(m => m.Address))}, Bcc: {string.Join(", ", message.Bcc.Select(m => m.Address))}, Subject: {message.Subject}, Body: {message.Body}");
            Log.Information($"[{mainCaseNumber}] [EXCEPT] SMTP Data: Host: {smtp.Host}, Port: {smtp.Port}, EnableSsl: {smtp.EnableSsl}, Credentials: {smtp.Credentials}, From: {message.From.Address}, To: {string.Join(", ", message.To.Select(m => m.Address))}, Bcc: {string.Join(", ", message.Bcc.Select(m => m.Address))}, Subject: {message.Subject}, Body: {message.Body}");
            smtp.Dispose();

            Log.Information($"[{mainCaseNumber}] [EXCEPT] Notification email sent to admin successfully.");
            Log.Information($"[{mainCaseNumber}] [EXCEPT] end of SendErrorNotification Function.");
        }