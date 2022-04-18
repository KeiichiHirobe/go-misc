ecs-taskをgoでrunする方法。

ちなみに、ecs-serviceのアプリ内でecs-taskをrunするには、以下のpolicyをecsのserviceのロールにつける必要があった。

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecs:RunTask"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": "iam:PassRole",
            "Resource": [
                "arn:aws:iam::xxxxxx:role/sample-ecs-task-role",
                "arn:aws:iam::xxxxxx:role/sample-ecs-task-exec-role"
            ],
            "Condition": {
                "StringLike": {
                    "iam:PassedToService": "ecs-tasks.amazonaws.com"
                }
            }
        }
    ]
}
```


