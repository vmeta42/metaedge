#!/bin/bash
app_dir=/usr/local/tomcat/webapps/
 
if [ "$(ls $app_dir/*.war)" ];then
    cd $app_dir
    for war in `ls *.war`;do
        war_name=${war%.war}
        mkdir $war_name
        mv $war $war_name/
        (cd $war_name/ && jar xf $war && rm -f $war)
    done
fi
