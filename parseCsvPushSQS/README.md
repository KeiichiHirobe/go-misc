This snippet are made for performance check for reading csv files and publish to sqs.

`read csv file and save to memory`: 50ms/100000 records,   500ms/1000000 records

`send message to sqs(not fifo) sequentially`: 183s/10000 records from my laptop,   107s/10000 records from EC2

`batch send(10 messages per request) to sqs(not fifo) sequentially`: 29s/10000 records from my laptop,   19s/10000 records from EC2

Additionally, Attaching VPC ENDPOINT did not get faster.

