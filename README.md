## Kubernetes y MongoDB replica set (en 14 comandos)

Prueba de concepto,  desplegar un cluster MongoDB en formato replica set (3 réplicas) sobre Kind Kubernetes. Darle "permanencia" a la BD. Además, implementar un "micro-servicio" para utilizar el cluster, accesible desde el host mediante Ingress.

La secuencia de comandos para la implementación está en el archivo comandos.txt. Unos comentarios:

- Empezamos por desplegar el cluster de Kubernetes usando Kind (línea 1 de comandos.txt), con 4 nodos, uno control-plane y tres workers. Sobre la configuración aplicada, config.yaml, más comentarios posteriormente.
- MongoDB ofrece imágenes para desplegar un cluster en Kubernetes. Aquí usamos instancias individuales de la BD y conformamos nosotros el replica set. La imagen para desplegar mongodb puede obtenerse de cualquier fuente disponible. En mi caso la bajé al repositorio local de Docker y se "pre-carga" en kind (línea 2)
- Creamos un secreto que contiene el usuario y el password para acceder a MongoDB (línea 3).
- Creamos un configmap (línea 4) que contiene un script que ejecutará cada una de las instancias de MongoDB al inicializarse. Esto conformará el replica set. Cuando este script se ejecute, cada instancia se declara como parte del replica set, que llamamos rs0 y fija los nombres de host que hará visibles (comando mongod --replSet rs0 ...) Por ejemplo, para la instancia mongo-2, hace visibles localhost y mongo-2 (tomado de la variable de ambiente {$HOSTNAME} que pasa el statefulset de Kubernetes al crear cada instancia). 
- Además, para la instancia mongo-0 se ejecuta el comando que inicializa el replica set con las tres instancias (comando mongo --eval ...). Este comando se ejecuta en mongo-0, pero en ese momento las otras instancias ya deben haberse creado y declarado como parte del cluster. Para lograrlo hacemos un loop esperando por esa condición.
- Cuando se inicialice el replica set deberán ser visibles, en el cluster de Kubernetes, los nombres de host mongo-1, mongo-2 y mongo-3. De ahí crear los servicios correspondientes (líneas 5 a 7) y estos deben ser de tipo ClusterIP.
- Estamos listos para desplegar los 3 pods con cada instancia de MongoDB. Lo hacemos mediante un statefulset (línea 8). Este crea 3 réplicas del container mongo, pasando las variables de entorno necesarios (root username y password). Monta además el configmap como una unidad del contenedor y ordena ejecutar el shell que contiene. Además agrega a cada pod una prueba de "liveness" ejecutando en el pod el comando de MongoDB que indica si la base de datos está disponible. En caso contrario el statefulset elimina ese pod y lo recrea.
- Sobre el almacenamiento de la BD. Bueno, debemos tratar que sea permanente, y que sea independiente del estado de cada instancia (pod), del replica set o incluso del cluster Kubernetes. Si no asignáramos almacenamiento alguno, el replica set sería inmune al fallo de un pod, o de nodos del cluster Kubernetes. Al recrearse el pod, este conectaría con el replica set y "reconstruiría" su copia de la BD. Esto lleva tiempo, no necesario si el almacenamiento es permanente. Y más grave aún, en caso de fallo del cluster Kubernetes perderíamos completamente la BD.
- En una situación real recurriríamos a un proveedor de almacenamiento permanente y dinámico, externo a nuestro cluster. Pero Kind, al ser una implementación de Kubernetes sobre Docker, no lo hace fácil (probé algunos proyectos de implementación de aprovisionador NFS y otros, sin resultado. Se agradece si alguien aporta una solución)
- Bueno, existe un truco (a lo mejor no muy "ortodoxo"). Kind ofrece una manera de "mapear" directorios de nuestra computadora a directorios de cada nodo del cluster Kubernetes. Ese es el sentido de las secciones extraMounts en el config.yaml con que se creó el cluster (línea 1). Allí se mapea el directorio /vdata de cada worker node (que es donde se ejecutarán los pods con instancias de MongoDB) a un directorio de la computadora. En mi caso F:\mongodb-cluster\data (estoy en Windows)
- Así cada nodo tendrá el directorio /vdata que apunta al directorio de nuestra computadora. Este directorio, y más importante aún, sus subdirectorios, están disponibles para montarse en los pods que se creen en el nodo. El truco se completa con el stateful (línea 8). Allí se declara un volumen vdata que se corresponde al directorio /vdata del nodo en que se crea y, en realidad, a un único directorio de nuestra computadora. Al crearse el pod, en el lugar que usa MongoDB para la BD (/data/db) se monta un subdirectorio de /vdata (se crea en ese momento), que tiene el nombre del pod (mediante subPathExpr: $(POD_NAME) y la declaracion de la variable de ambiente POD_NAME, que es el nombre que asigna el stateful a cada pod) Luego, al crearse un pod, por ejemplo mongo-1, en ese nodo habrá un directorio /vdata/mongo-1, que en realidad está en nuestra computadora, en mi caso, F:\mongodb-cluster\data\mongo-1. Al crearse los tres pods, tendremos tres subdirectorios (mongo-0, mongo-1, mongo-2) en nuestra computadora, uno para cada una de las BD de las tres instancias del replica set (obviamente, el uso de subdirectorios es necesario, para que cada instancia tenga su almacenamiento independiente)
- Si un pod termina y se recrea puede caer en otro nodo del cluster Kubernetes, pero esto no importa, porque la referencia siempre llevará al correspondiente subdirectorio de nuestra computadora. Si termina y se reinicia el cluster Kubernetes, los subdirectorios y su contenido permanecen, nuestra BD es permanente (eso espero).
- Tener una base de datos en replica set sin poder probarla y "jugar" con ella no es nada. Así que creamos un "micro-servicio" en Go que la utiliza (el código en go-client). Es muy sencillo, expone en el puerto 8080 una mini-interfase "REST". Con un GET lista los nombres de los miembros de la colección personas de la base de datos prueba. Con un POST crea una persona (sólo registra el nombre) y de paso, la primera vez, crea la base de datos y la colección. En ese directorio está también el DockerFile para crear la imagen go-mongo-client que cargamos en Kind (línea 9)
- Creamos un deployment con dos réplicas (pods) de nuestro micro-servicio (línea 10) y para que sea accesible en el cluster creamos el correspondiente servicio (línea 11)
- Para hacerlo accesible en nuestra computadora, y balancear entre las dos réplicas, desplegamos Ingress en el cluster (pasos 12, 13 y 14) siguiendo las indicaciones de Kind (que necesita la cláusula extraPortMappings en el config.yaml usado para crear el cluster)
- Ahora podemos probar:
```
    curl -X POST localhost/Juan
    OK

    curl -X POST localhost/Pepe
    OK

    curl -X POST localhost/Roberto
    OK

    curl -X GET localhost
    [{Juan} {Pepe} {Roberto}]
```
- Claro, resulta interesante probar a eliminar un pod:
```
    kubectl delete pod mongo-2
```
- O todos los pods:
```
    kubectl delete statefulset mongo
```

y los relanzamos con:

```    
    kubectl apply -f mongo-sfs.yaml
```
- En estos casos la base de datos permanece operativa y se reconstruye (a lo mejor una petición demora algo mientras se reconfigura el cluster y el réplica set de MongoDB)
- O terminar el cluster Kubernetes y volverlo a desplegar para comprobar que la BD es permanente y no se pierde con esto.
- En realidad 14 comandos son muchos. Los archivos yaml se pueden consolidar, hacer scripts. Si lo intentan, por favor, déjenme saber en avaco.digital@ gmail.com

## Actualización (desplegar con un solo un comando)

Se adiciona un archivo batch para desplegar todo con un solo comando, comandos.bat (para Windows aunque debe funcionar con pequeños cambios en Linux o Mac). Utiliza el archivo yaml "concentrado" completo.yaml

  



