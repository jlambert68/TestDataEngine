On Windows, from the repo root in PowerShell:
.\build.ps1

That will:
• run npm ci in ui\
• build ui\dist
• build:
◦ bin\testdataengine.exe
◦ bin\testdataengine-web.exe
◦ bin\csv2sqlite.exe

Optional flags:
.\build.ps1 -SkipUi
.\build.ps1 -SkipGo
.\build.ps1 -OutputDir out

After that, run the web app with:
$env:HTTP_ADDR=":8080"
.\bin\testdataengine-web.exe

and open http://localhost:8080.

Important detail: testdataengine-web.exe serves files from ui\dist, so keep the repo layout intact when running it.
