@echo off
cd /d E:\Micha\.LocalCode\ArmorChat
echo ===================================
echo Running Gradle Build...
echo ===================================
call gradlew.bat clean assembleDebug --no-daemon 2>&1
echo ===================================
echo Build Complete
echo ===================================
