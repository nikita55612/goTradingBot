import argparse
import server
import time


if __name__ == '__main__':
    parser = argparse.ArgumentParser()

    parser.add_argument('-H', default='localhost')
    parser.add_argument('-P', default=8080, type=int)

    args = parser.parse_args().__dict__
    host = args.get('H')
    port = args.get('P')

    try:
        server.run(host, port)
    except Exception as e:
        print(e)
        time.sleep(5)
        server.run(host, port)
