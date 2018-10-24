import json
import requests
import functions


class Config(object):
    def __init__(self):
        self.filename = "./.config.json"

    def read(self):
        with open(self.filename, 'r') as f:
            return json.load(f)
    
    def write(self, conf):
        with open(self.filename, 'w') as f:
            json.dump(conf, f)

class Image(object):
    _dumpable = True
    def __init__(self, blob, img, transitions):
        self.repo_tags = img['RepoTags']
        self.sha = img['ImageSha']
        self.priority = img['Priority']
        self.transitions = transitions
        self.state = img['ScanStatus']
        self.scan_results = img['ScanResults']

class Container(object):
    _dumpable = True
    def __init__(self, blob, cont, image):
        self.name = cont['Name']
        self.image = image

class Pod(object):
    _dumpable = True
    def __init__(self, blob, pod, images):
        self.name = pod['Name']
        self.namespace = pod['Namespace']
        self.containers = []
        for c in pod['Containers']:
            self.containers.append(Container(blob, c, images.get(c['Image']['Sha'], None)))

class Transition(object):
    _dumpable = True
    def __init__(self, blob, tran):
        self.sha = tran['Sha']
        self.from_state = tran['From']
        self.to_state = tran['To']
        self.err = tran['Err']
        self.time = tran['Time']

class Dump(object):
    _dumpable = True
    def __init__(self, blob):
        self.images = {}
        self.transitions = {}
        for t in blob['CoreModel']['ImageTransitions']:
            if t['Sha'] not in self.transitions:
                self.transitions[t['Sha']] = []
            self.transitions[t['Sha']].append(Transition(blob, t))
        for (sha, img) in blob['CoreModel']['Images'].items():
            self.images[sha] = Image(blob, img, self.transitions.get(sha, []))
        self.pods = {}
        for (name, p) in blob['CoreModel']['Pods'].items():
            self.pods[name] = Pod(blob, p, self.images)
        self.ns = {}
        for (name, p) in self.pods.iteritems():
            ns = p.namespace
            if ns not in self.ns:
                self.ns[ns] = []
            self.ns[ns].append(p)

    def namespace_images(self, namespace):
        pods = self.ns.get(namespace, [])
        images = set()
        for pod in pods:
            images = images.union(c.image.sha for c in pod.containers)
        return images
    
    def scan_queue(self):
        pass
    
    def scan_statuses(self):
        pass

def dump(obj):
    if isinstance(obj, list):
        return map(dump, obj)
    if isinstance(obj, dict):
        return dict((key, dump(value)) for (key, value) in obj.iteritems())
    if not hasattr(obj, '_dumpable'):
        print "not dumpable,", type(obj)
        return obj
    print "dumpable,", type(obj)
    out = {}
    for name in dir(obj):
        if isinstance(getattr(obj, name), type(Config.read)):
            continue
        if 'a' <= name[0] <= 'z':
            out[name] = dump(getattr(obj, name))
    return out

def main():
    import sys
    conf = Config()
    if len(sys.argv) > 2:
        url, command = sys.argv[1:3]
    else:
        url = conf.read()['url']
        command = sys.argv[1]
    conf.write({'url': url})
    resp = requests.get(url)
#    print dir(resp)
#    print resp.json()['CoreModel'].keys()
    d = Dump(resp.json())
#    print d.images, d.pods, dir(d), type(d)
#    print json.dumps(dump(d), indent=2)
#    print dir(functions)
#    print functions.get_pod_images(d)
#    print '\n'.join(functions.IndentedList(functions.get_pod_images(d)).lines())
    if command == "namespace_pod_images":
        print functions.namespace_pod_images(d)#sys.argv[2])
    elif command == "namespace_images":
        print functions.namespace_images(d)

main()
