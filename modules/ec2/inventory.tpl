[${role}.hosts]
%{ for host in hosts ~}
${host.tags.Name} = { ansible_host = '${host.public_ip}', private_ip = '${host.private_ip}' }
%{ endfor ~}

[${role}.vars]
ansible_user = '${ssh_user}'
